package api

import (
	"strings"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/services"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

// UserLLMSettingsApi represents user large language model settings api
type UserLLMSettingsApi struct {
	userLLMSettings *services.UserLLMSettingsService
	users           *services.UserService
}

// Initialize a user large language model settings api singleton instance
var (
	UserLLMSettings = &UserLLMSettingsApi{
		userLLMSettings: services.UserLLMSettings,
		users:           services.Users,
	}
)

// UserLLMSettingsGetHandler returns large language model settings of current user
func (a *UserLLMSettingsApi) UserLLMSettingsGetHandler(c *core.WebContext) (any, *errs.Error) {
	uid := c.GetCurrentUid()

	userLLMSetting, err := a.userLLMSettings.GetUserLLMSettingByUid(c, uid)

	if err != nil {
		log.Errorf(c, "[user_llm_settings.UserLLMSettingsGetHandler] failed to get user large language model setting for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	return userLLMSetting.ToUserLLMSettingResponse(), nil
}

// UserLLMSettingsUpdateHandler updates large language model settings of current user
func (a *UserLLMSettingsApi) UserLLMSettingsUpdateHandler(c *core.WebContext) (any, *errs.Error) {
	var userLLMSettingUpdateReq models.UserLLMSettingUpdateRequest
	err := c.ShouldBindJSON(&userLLMSettingUpdateReq)

	if err != nil {
		log.Warnf(c, "[user_llm_settings.UserLLMSettingsUpdateHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	uid := c.GetCurrentUid()
	user, err := a.users.GetUserById(c, uid)

	if err != nil {
		if !errs.IsCustomError(err) {
			log.Warnf(c, "[user_llm_settings.UserLLMSettingsUpdateHandler] failed to get user for user \"uid:%d\", because %s", uid, err.Error())
		}

		return false, errs.ErrUserNotFound
	}

	if user.FeatureRestriction.Contains(core.USER_FEATURE_RESTRICTION_TYPE_CREATE_TRANSACTION_FROM_AI_IMAGE_RECOGNITION) {
		return false, errs.ErrNotPermittedToPerformThisAction
	}

	currentUserLLMSetting, err := a.userLLMSettings.GetUserLLMSettingByUid(c, uid)

	if err != nil {
		log.Errorf(c, "[user_llm_settings.UserLLMSettingsUpdateHandler] failed to get current user large language model setting for user \"uid:%d\", because %s", uid, err.Error())
		return false, errs.Or(err, errs.ErrOperationFailed)
	}

	userLLMSetting := a.createUserLLMSettingModel(uid, &userLLMSettingUpdateReq, currentUserLLMSetting)

	if userLLMSetting.Enabled {
		if validationErr := validateUserLLMSetting(userLLMSetting); validationErr != nil {
			return false, validationErr
		}
	}

	if err := a.userLLMSettings.UpdateUserLLMSetting(c, userLLMSetting); err != nil {
		log.Errorf(c, "[user_llm_settings.UserLLMSettingsUpdateHandler] failed to update user large language model setting for user \"uid:%d\", because %s", uid, err.Error())
		return false, errs.Or(err, errs.ErrOperationFailed)
	}

	return userLLMSetting.ToUserLLMSettingResponse(), nil
}

func (a *UserLLMSettingsApi) createUserLLMSettingModel(uid int64, req *models.UserLLMSettingUpdateRequest, currentSetting *models.UserLLMSetting) *models.UserLLMSetting {
	userLLMSetting := &models.UserLLMSetting{
		Uid:                                 uid,
		Enabled:                             req.Enabled,
		LLMProvider:                         strings.TrimSpace(req.LLMProvider),
		OpenAIAPIKey:                        strings.TrimSpace(req.OpenAIAPIKey),
		OpenAIModelID:                       strings.TrimSpace(req.OpenAIModelID),
		OpenAICompatibleBaseURL:             strings.TrimSpace(req.OpenAICompatibleBaseURL),
		OpenAICompatibleAPIKey:              strings.TrimSpace(req.OpenAICompatibleAPIKey),
		OpenAICompatibleModelID:             strings.TrimSpace(req.OpenAICompatibleModelID),
		OpenRouterAPIKey:                    strings.TrimSpace(req.OpenRouterAPIKey),
		OpenRouterModelID:                   strings.TrimSpace(req.OpenRouterModelID),
		OllamaServerURL:                     strings.TrimSpace(req.OllamaServerURL),
		OllamaModelID:                       strings.TrimSpace(req.OllamaModelID),
		GoogleAIAPIKey:                      strings.TrimSpace(req.GoogleAIAPIKey),
		GoogleAIModelID:                     strings.TrimSpace(req.GoogleAIModelID),
		LargeLanguageModelAPIRequestTimeout: req.LargeLanguageModelAPIRequestTimeout,
		LargeLanguageModelAPIProxy:          strings.TrimSpace(req.LargeLanguageModelAPIProxy),
		LargeLanguageModelAPISkipTLSVerify:  req.LargeLanguageModelAPISkipTLSVerify,
	}

	if userLLMSetting.LargeLanguageModelAPIProxy == "" {
		userLLMSetting.LargeLanguageModelAPIProxy = "system"
	}

	if currentSetting == nil {
		return userLLMSetting
	}

	if userLLMSetting.OpenAIAPIKey == "" {
		userLLMSetting.OpenAIAPIKey = currentSetting.OpenAIAPIKey
	}

	if userLLMSetting.OpenAICompatibleAPIKey == "" {
		userLLMSetting.OpenAICompatibleAPIKey = currentSetting.OpenAICompatibleAPIKey
	}

	if userLLMSetting.OpenRouterAPIKey == "" {
		userLLMSetting.OpenRouterAPIKey = currentSetting.OpenRouterAPIKey
	}

	if userLLMSetting.GoogleAIAPIKey == "" {
		userLLMSetting.GoogleAIAPIKey = currentSetting.GoogleAIAPIKey
	}

	return userLLMSetting
}

func validateUserLLMSetting(userLLMSetting *models.UserLLMSetting) *errs.Error {
	if userLLMSetting.LLMProvider != settings.OpenAILLMProvider &&
		userLLMSetting.LLMProvider != settings.OpenAICompatibleLLMProvider &&
		userLLMSetting.LLMProvider != settings.OpenRouterLLMProvider &&
		userLLMSetting.LLMProvider != settings.OllamaLLMProvider &&
		userLLMSetting.LLMProvider != settings.GoogleAILLMProvider {
		return errs.ErrIncompleteOrIncorrectSubmission
	}

	if userLLMSetting.LLMProvider == settings.OpenAILLMProvider && (userLLMSetting.OpenAIAPIKey == "" || userLLMSetting.OpenAIModelID == "") {
		return errs.ErrIncompleteOrIncorrectSubmission
	}

	if userLLMSetting.LLMProvider == settings.OpenAICompatibleLLMProvider && (userLLMSetting.OpenAICompatibleBaseURL == "" || userLLMSetting.OpenAICompatibleModelID == "") {
		return errs.ErrIncompleteOrIncorrectSubmission
	}

	if userLLMSetting.LLMProvider == settings.OpenRouterLLMProvider && (userLLMSetting.OpenRouterAPIKey == "" || userLLMSetting.OpenRouterModelID == "") {
		return errs.ErrIncompleteOrIncorrectSubmission
	}

	if userLLMSetting.LLMProvider == settings.OllamaLLMProvider && (userLLMSetting.OllamaServerURL == "" || userLLMSetting.OllamaModelID == "") {
		return errs.ErrIncompleteOrIncorrectSubmission
	}

	if userLLMSetting.LLMProvider == settings.GoogleAILLMProvider && (userLLMSetting.GoogleAIAPIKey == "" || userLLMSetting.GoogleAIModelID == "") {
		return errs.ErrIncompleteOrIncorrectSubmission
	}

	return nil
}
