package api

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/mayswind/financially_free/pkg/core"
	"github.com/mayswind/financially_free/pkg/errs"
	"github.com/mayswind/financially_free/pkg/llm"
	"github.com/mayswind/financially_free/pkg/llm/data"
	"github.com/mayswind/financially_free/pkg/log"
	"github.com/mayswind/financially_free/pkg/models"
	"github.com/mayswind/financially_free/pkg/services"
	"github.com/mayswind/financially_free/pkg/settings"
	"github.com/mayswind/financially_free/pkg/templates"
	"github.com/mayswind/financially_free/pkg/utils"
)

// LargeLanguageModelsApi represents large language models api
type LargeLanguageModelsApi struct {
	ApiUsingConfig
	transactionCategories *services.TransactionCategoryService
	transactionTags       *services.TransactionTagService
	accounts              *services.AccountService
	users                 *services.UserService
	userLLMSettings       *services.UserLLMSettingsService
}

// Initialize a large language models api singleton instance
var (
	LargeLanguageModels = &LargeLanguageModelsApi{
		ApiUsingConfig: ApiUsingConfig{
			container: settings.Container,
		},
		transactionCategories: services.TransactionCategories,
		transactionTags:       services.TransactionTags,
		accounts:              services.Accounts,
		users:                 services.Users,
		userLLMSettings:       services.UserLLMSettings,
	}
)

type transactionRecognitionContext struct {
	accountMap            map[string]*models.Account
	accountNames          []string
	expenseCategoryMap    map[string]*models.TransactionCategory
	expenseCategoryNames  []string
	incomeCategoryMap     map[string]*models.TransactionCategory
	incomeCategoryNames   []string
	transferCategoryMap   map[string]*models.TransactionCategory
	transferCategoryNames []string
	tagMap                map[string]*models.TransactionTag
	tagNames              []string
}

// RecognizeReceiptImageHandler returns the recognized receipt image result
func (a *LargeLanguageModelsApi) RecognizeReceiptImageHandler(c *core.WebContext) (any, *errs.Error) {
	clientTimezone, err := c.GetClientTimezone()

	if err != nil {
		log.Warnf(c, "[large_language_models.RecognizeReceiptImageHandler] cannot get client timezone, because %s", err.Error())
		return nil, errs.ErrClientTimezoneOffsetInvalid
	}

	uid := c.GetCurrentUid()
	user, err := a.users.GetUserById(c, uid)

	if err != nil {
		if !errs.IsCustomError(err) {
			log.Warnf(c, "[large_language_models.RecognizeReceiptImageHandler] failed to get user for user \"uid:%d\", because %s", uid, err.Error())
		}

		return false, errs.ErrUserNotFound
	}

	if user.FeatureRestriction.Contains(core.USER_FEATURE_RESTRICTION_TYPE_CREATE_TRANSACTION_FROM_AI_IMAGE_RECOGNITION) {
		return false, errs.ErrNotPermittedToPerformThisAction
	}

	llmConfig, recognitionErr := a.getEffectiveLLMConfig(c, uid)

	if recognitionErr != nil {
		return nil, recognitionErr
	}

	form, err := c.MultipartForm()

	if err != nil {
		log.Errorf(c, "[large_language_models.RecognizeReceiptImageHandler] failed to get multi-part form data for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.ErrParameterInvalid
	}

	imageFiles := form.File["image"]

	if len(imageFiles) < 1 {
		log.Warnf(c, "[large_language_models.RecognizeReceiptImageHandler] there is no image in request for user \"uid:%d\"", uid)
		return nil, errs.ErrNoAIRecognitionImage
	}

	if imageFiles[0].Size < 1 {
		log.Warnf(c, "[large_language_models.RecognizeReceiptImageHandler] the size of image in request is zero for user \"uid:%d\"", uid)
		return nil, errs.ErrAIRecognitionImageIsEmpty
	}

	if imageFiles[0].Size > int64(a.CurrentConfig().MaxAIRecognitionPictureFileSize) {
		log.Warnf(c, "[large_language_models.RecognizeReceiptImageHandler] the upload file size \"%d\" exceeds the maximum size \"%d\" of image for user \"uid:%d\"", imageFiles[0].Size, a.CurrentConfig().MaxAIRecognitionPictureFileSize, uid)
		return nil, errs.ErrExceedMaxAIRecognitionImageFileSize
	}

	fileExtension := utils.GetFileNameExtension(imageFiles[0].Filename)
	contentType := utils.GetImageContentType(fileExtension)

	if contentType == "" {
		log.Warnf(c, "[large_language_models.RecognizeReceiptImageHandler] the file extension \"%s\" of image in request is not supported for user \"uid:%d\"", fileExtension, uid)
		return nil, errs.ErrImageTypeNotSupported
	}

	imageFile, err := imageFiles[0].Open()

	if err != nil {
		log.Errorf(c, "[large_language_models.RecognizeReceiptImageHandler] failed to get image file from request for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.ErrOperationFailed
	}

	defer imageFile.Close()

	imageData, err := io.ReadAll(imageFile)

	if err != nil {
		log.Errorf(c, "[large_language_models.RecognizeReceiptImageHandler] failed to read image file from request for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.ErrOperationFailed
	}

	recognitionContext, recognitionErr := a.getTransactionRecognitionContext(c, uid)

	if recognitionErr != nil {
		return nil, recognitionErr
	}

	systemPrompt, recognitionErr := a.buildTransactionRecognitionSystemPrompt(c, uid, clientTimezone, templates.SYSTEM_PROMPT_RECEIPT_IMAGE_RECOGNITION, recognitionContext)

	if recognitionErr != nil {
		return nil, recognitionErr
	}

	llmRequest := &data.LargeLanguageModelRequest{
		Stream:                false,
		SystemPrompt:          systemPrompt,
		UserPrompt:            imageData,
		UserPromptType:        data.LARGE_LANGUAGE_MODEL_REQUEST_PROMPT_TYPE_IMAGE_URL,
		UserPromptContentType: contentType,
	}

	llmResponse, err := llm.Container.GetJsonResponseByModel(c, c.GetCurrentUid(), llmConfig, llmRequest)

	if err != nil {
		log.Errorf(c, "[large_language_models.RecognizeReceiptImageHandler] failed to get llm response user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	if llmResponse == nil || len(strings.TrimSpace(llmResponse.Content)) == 0 || strings.HasPrefix(strings.TrimSpace(llmResponse.Content), "{}") {
		return nil, errs.ErrNoTransactionInformationInImage
	}

	var result *models.RecognizedTransactionResult

	if err := json.Unmarshal([]byte(strings.TrimSpace(llmResponse.Content)), &result); err != nil {
		log.Errorf(c, "[large_language_models.RecognizeReceiptImageHandler] failed to unmarshal recognized receipt image result from llm response \"%s\" for user \"uid:%d\", because %s", llmResponse.Content, uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	return a.parseRecognizedTransactionResponse(c, uid, clientTimezone, result, recognitionContext)
}

// RecognizeTransactionTextHandler returns the recognized transaction text result
func (a *LargeLanguageModelsApi) RecognizeTransactionTextHandler(c *core.WebContext) (any, *errs.Error) {
	clientTimezone, err := c.GetClientTimezone()

	if err != nil {
		log.Warnf(c, "[large_language_models.RecognizeTransactionTextHandler] cannot get client timezone, because %s", err.Error())
		return nil, errs.ErrClientTimezoneOffsetInvalid
	}

	var textReq models.RecognizedTransactionTextRequest

	if err := c.ShouldBindJSON(&textReq); err != nil {
		log.Warnf(c, "[large_language_models.RecognizeTransactionTextHandler] parse request failed, because %s", err.Error())
		return nil, errs.ErrParameterInvalid
	}

	uid := c.GetCurrentUid()
	user, err := a.users.GetUserById(c, uid)

	if err != nil {
		if !errs.IsCustomError(err) {
			log.Warnf(c, "[large_language_models.RecognizeTransactionTextHandler] failed to get user for user \"uid:%d\", because %s", uid, err.Error())
		}

		return false, errs.ErrUserNotFound
	}

	if user.FeatureRestriction.Contains(core.USER_FEATURE_RESTRICTION_TYPE_CREATE_TRANSACTION_FROM_AI_IMAGE_RECOGNITION) {
		return false, errs.ErrNotPermittedToPerformThisAction
	}

	llmConfig, recognitionErr := a.getEffectiveLLMConfig(c, uid)

	if recognitionErr != nil {
		return nil, recognitionErr
	}

	recognitionContext, recognitionErr := a.getTransactionRecognitionContext(c, uid)

	if recognitionErr != nil {
		return nil, recognitionErr
	}

	systemPrompt, recognitionErr := a.buildTransactionRecognitionSystemPrompt(c, uid, clientTimezone, templates.SYSTEM_PROMPT_TRANSACTION_TEXT_RECOGNITION, recognitionContext)

	if recognitionErr != nil {
		return nil, recognitionErr
	}

	llmRequest := &data.LargeLanguageModelRequest{
		Stream:         false,
		SystemPrompt:   systemPrompt,
		UserPrompt:     []byte(textReq.Text),
		UserPromptType: data.LARGE_LANGUAGE_MODEL_REQUEST_PROMPT_TYPE_TEXT,
	}

	llmResponse, err := llm.Container.GetJsonResponseByModel(c, uid, llmConfig, llmRequest)

	if err != nil {
		log.Errorf(c, "[large_language_models.RecognizeTransactionTextHandler] failed to get llm response user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	if llmResponse == nil || len(strings.TrimSpace(llmResponse.Content)) == 0 || strings.HasPrefix(strings.TrimSpace(llmResponse.Content), "{}") {
		return nil, errs.ErrNoTransactionInformationInImage
	}

	var result *models.RecognizedTransactionResult

	if err := json.Unmarshal([]byte(strings.TrimSpace(llmResponse.Content)), &result); err != nil {
		log.Errorf(c, "[large_language_models.RecognizeTransactionTextHandler] failed to unmarshal recognized transaction text result from llm response \"%s\" for user \"uid:%d\", because %s", llmResponse.Content, uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	return a.parseRecognizedTransactionResponse(c, uid, clientTimezone, result, recognitionContext)
}

func (a *LargeLanguageModelsApi) getEffectiveLLMConfig(c core.Context, uid int64) (*settings.LLMConfig, *errs.Error) {
	userLLMSetting, err := a.userLLMSettings.GetUserLLMSettingByUid(c, uid)

	if err != nil {
		log.Errorf(c, "[large_language_models.getEffectiveLLMConfig] failed to get user large language model setting for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	if userLLMSetting != nil && userLLMSetting.Enabled {
		llmConfig := userLLMSetting.ToLLMConfig(a.CurrentConfig().ReceiptImageRecognitionLLMConfig)

		if llmConfig != nil && llmConfig.LLMProvider != "" {
			return llmConfig, nil
		}
	}

	if a.CurrentConfig().ReceiptImageRecognitionLLMConfig != nil &&
		a.CurrentConfig().ReceiptImageRecognitionLLMConfig.LLMProvider != "" &&
		a.CurrentConfig().TransactionFromAIImageRecognition {
		return a.CurrentConfig().ReceiptImageRecognitionLLMConfig, nil
	}

	return nil, errs.ErrLargeLanguageModelProviderNotEnabled
}

func (a *LargeLanguageModelsApi) getTransactionRecognitionContext(c core.Context, uid int64) (*transactionRecognitionContext, *errs.Error) {
	accounts, err := a.accounts.GetAllAccountsByUid(c, uid)

	if err != nil {
		log.Errorf(c, "[large_language_models.getTransactionRecognitionContext] failed to get all accounts for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	recognitionContext := &transactionRecognitionContext{
		accountMap:            a.accounts.GetVisibleAccountNameMapByList(accounts),
		accountNames:          make([]string, 0, len(accounts)),
		expenseCategoryMap:    make(map[string]*models.TransactionCategory),
		expenseCategoryNames:  make([]string, 0),
		incomeCategoryMap:     make(map[string]*models.TransactionCategory),
		incomeCategoryNames:   make([]string, 0),
		transferCategoryMap:   make(map[string]*models.TransactionCategory),
		transferCategoryNames: make([]string, 0),
	}

	for i := 0; i < len(accounts); i++ {
		if accounts[i].Hidden || accounts[i].Type == models.ACCOUNT_TYPE_MULTI_SUB_ACCOUNTS {
			continue
		}

		recognitionContext.accountNames = append(recognitionContext.accountNames, accounts[i].Name)
	}

	categories, err := a.transactionCategories.GetAllCategoriesByUid(c, uid, 0, -1)

	if err != nil {
		log.Errorf(c, "[large_language_models.getTransactionRecognitionContext] failed to get categories for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	for i := 0; i < len(categories); i++ {
		category := categories[i]

		if category.Hidden || category.ParentCategoryId == models.LevelOneTransactionCategoryParentId {
			continue
		}

		if category.Type == models.CATEGORY_TYPE_INCOME {
			recognitionContext.incomeCategoryMap[category.Name] = category
			recognitionContext.incomeCategoryNames = append(recognitionContext.incomeCategoryNames, category.Name)
		} else if category.Type == models.CATEGORY_TYPE_EXPENSE {
			recognitionContext.expenseCategoryMap[category.Name] = category
			recognitionContext.expenseCategoryNames = append(recognitionContext.expenseCategoryNames, category.Name)
		} else if category.Type == models.CATEGORY_TYPE_TRANSFER {
			recognitionContext.transferCategoryMap[category.Name] = category
			recognitionContext.transferCategoryNames = append(recognitionContext.transferCategoryNames, category.Name)
		}
	}

	tags, err := a.transactionTags.GetAllTagsByUid(c, uid)

	if err != nil {
		log.Errorf(c, "[large_language_models.getTransactionRecognitionContext] failed to get tags for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	recognitionContext.tagMap = a.transactionTags.GetVisibleTagNameMapByList(tags)
	recognitionContext.tagNames = make([]string, 0, len(tags))

	for i := 0; i < len(tags); i++ {
		if tags[i].Hidden {
			continue
		}

		recognitionContext.tagNames = append(recognitionContext.tagNames, tags[i].Name)
	}

	return recognitionContext, nil
}

func (a *LargeLanguageModelsApi) buildTransactionRecognitionSystemPrompt(c core.Context, uid int64, clientTimezone *time.Location, templateName templates.KnownTemplate, recognitionContext *transactionRecognitionContext) (string, *errs.Error) {
	systemPrompt, err := templates.GetTemplate(templateName)

	if err != nil {
		log.Errorf(c, "[large_language_models.buildTransactionRecognitionSystemPrompt] failed to get system prompt template for user \"uid:%d\", because %s", uid, err.Error())
		return "", errs.Or(err, errs.ErrOperationFailed)
	}

	systemPromptParams := map[string]any{
		"CurrentDateTime":          utils.FormatUnixTimeToLongDateTime(time.Now().Unix(), clientTimezone),
		"AllExpenseCategoryNames":  strings.Join(recognitionContext.expenseCategoryNames, "\n"),
		"AllIncomeCategoryNames":   strings.Join(recognitionContext.incomeCategoryNames, "\n"),
		"AllTransferCategoryNames": strings.Join(recognitionContext.transferCategoryNames, "\n"),
		"AllAccountNames":          strings.Join(recognitionContext.accountNames, "\n"),
		"AllTagNames":              strings.Join(recognitionContext.tagNames, "\n"),
	}

	var bodyBuffer bytes.Buffer
	err = systemPrompt.Execute(&bodyBuffer, systemPromptParams)

	if err != nil {
		log.Errorf(c, "[large_language_models.buildTransactionRecognitionSystemPrompt] failed to get final system prompt from template for user \"uid:%d\", because %s", uid, err.Error())
		return "", errs.Or(err, errs.ErrOperationFailed)
	}

	return strings.ReplaceAll(bodyBuffer.String(), "\r\n", "\n"), nil
}

func (a *LargeLanguageModelsApi) parseRecognizedTransactionResponse(c core.Context, uid int64, clientTimezone *time.Location, recognizedResult *models.RecognizedTransactionResult, recognitionContext *transactionRecognitionContext) (*models.RecognizedTransactionResponse, *errs.Error) {
	recognizedTransactionResponse := &models.RecognizedTransactionResponse{
		Type: models.TRANSACTION_TYPE_EXPENSE,
	}

	if recognizedResult == nil {
		log.Errorf(c, "[large_language_models.parseRecognizedTransactionResponse] recognized result is null")
		return nil, errs.ErrNoTransactionInformationInImage
	}

	if recognizedResult.Type == "income" {
		recognizedTransactionResponse.Type = models.TRANSACTION_TYPE_INCOME

		if len(recognizedResult.CategoryName) > 0 {
			category, exists := recognitionContext.incomeCategoryMap[recognizedResult.CategoryName]

			if exists {
				recognizedTransactionResponse.CategoryId = category.CategoryId
			}
		}
	} else if recognizedResult.Type == "expense" {
		recognizedTransactionResponse.Type = models.TRANSACTION_TYPE_EXPENSE

		if len(recognizedResult.CategoryName) > 0 {
			category, exists := recognitionContext.expenseCategoryMap[recognizedResult.CategoryName]

			if exists {
				recognizedTransactionResponse.CategoryId = category.CategoryId
			}
		}
	} else if recognizedResult.Type == "transfer" {
		recognizedTransactionResponse.Type = models.TRANSACTION_TYPE_TRANSFER

		if len(recognizedResult.CategoryName) > 0 {
			category, exists := recognitionContext.transferCategoryMap[recognizedResult.CategoryName]

			if exists {
				recognizedTransactionResponse.CategoryId = category.CategoryId
			}
		}
	} else if len(recognizedResult.Type) == 0 {
		return nil, errs.ErrNoTransactionInformationInImage
	} else {
		log.Errorf(c, "[large_language_models.parseRecognizedTransactionResponse] recognized transaction type \"%s\" is invalid", recognizedResult.Type)
		return nil, errs.ErrOperationFailed
	}

	if len(recognizedResult.Time) > 0 {
		longDateTime := a.getLongDateTime(recognizedResult.Time)
		timestamp, err := utils.ParseFromLongDateTimeInTimeZone(longDateTime, clientTimezone)

		if err != nil {
			log.Warnf(c, "[large_language_models.parseRecognizedTransactionResponse] recognized time \"%s\" is invalid", recognizedResult.Time)
		} else {
			recognizedTransactionResponse.Time = timestamp.Unix()
		}
	}

	if len(recognizedResult.Amount) > 0 {
		amount, err := utils.ParseAmount(recognizedResult.Amount)

		if err != nil {
			log.Errorf(c, "[large_language_models.parseRecognizedTransactionResponse] recognized amount \"%s\" is invalid", recognizedResult.Amount)
			return nil, errs.ErrOperationFailed
		}

		recognizedTransactionResponse.SourceAmount = amount

		if recognizedTransactionResponse.Type == models.TRANSACTION_TYPE_TRANSFER && len(recognizedResult.DestinationAmount) > 0 {
			destinationAmount, err := utils.ParseAmount(recognizedResult.DestinationAmount)

			if err != nil {
				log.Errorf(c, "[large_language_models.parseRecognizedTransactionResponse] recognized destination amount \"%s\" is invalid", recognizedResult.DestinationAmount)
				return nil, errs.ErrOperationFailed
			}

			recognizedTransactionResponse.DestinationAmount = destinationAmount
		}
	}

	if len(recognizedResult.AccountName) > 0 {
		account, exists := recognitionContext.accountMap[recognizedResult.AccountName]

		if exists {
			recognizedTransactionResponse.SourceAccountId = account.AccountId
		}
	}

	if len(recognizedResult.DestinationAccountName) > 0 {
		account, exists := recognitionContext.accountMap[recognizedResult.DestinationAccountName]

		if exists {
			recognizedTransactionResponse.DestinationAccountId = account.AccountId
		}
	}

	if len(recognizedResult.TagNames) > 0 {
		tagIds := make([]string, 0, len(recognizedResult.TagNames))

		for i := 0; i < len(recognizedResult.TagNames); i++ {
			tagName := recognizedResult.TagNames[i]
			tag, exists := recognitionContext.tagMap[tagName]

			if exists {
				tagIds = append(tagIds, utils.Int64ToString(tag.TagId))
			}
		}

		recognizedTransactionResponse.TagIds = tagIds
	}

	if len(recognizedResult.Description) > 0 {
		recognizedTransactionResponse.Comment = recognizedResult.Description
	}

	return recognizedTransactionResponse, nil
}

func (a *LargeLanguageModelsApi) getLongDateTime(dateTime string) string {
	if utils.IsValidLongDateTimeFormat(dateTime) {
		return dateTime
	}

	if utils.IsValidLongDateTimeWithoutSecondFormat(dateTime) {
		return dateTime + ":00"
	}

	if utils.IsValidLongDateFormat(dateTime) {
		return dateTime + " 00:00:00"
	}

	return dateTime
}
