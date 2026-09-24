package api

import (
	"strings"

	"github.com/mayswind/financially_free/pkg/core"
	"github.com/mayswind/financially_free/pkg/duplicatechecker"
	"github.com/mayswind/financially_free/pkg/errs"
	"github.com/mayswind/financially_free/pkg/log"
	"github.com/mayswind/financially_free/pkg/models"
	"github.com/mayswind/financially_free/pkg/services"
	"github.com/mayswind/financially_free/pkg/settings"
	"github.com/mayswind/financially_free/pkg/utils"
)

type SocialGiftsApi struct {
	ApiUsingDuplicateChecker
	socialGifts  *services.SocialGiftService
	transactions *services.TransactionService
	users        *services.UserService
}

var SocialGifts = &SocialGiftsApi{
	ApiUsingDuplicateChecker: ApiUsingDuplicateChecker{
		ApiUsingConfig: ApiUsingConfig{container: settings.Container},
		container:      duplicatechecker.Container,
	},
	socialGifts:  services.SocialGifts,
	transactions: services.Transactions,
	users:        services.Users,
}

func validateSocialGiftMetadata(direction, eventType, eventName string) *errs.Error {
	if !services.IsValidSocialGiftDirection(direction) {
		return errs.ErrSocialGiftDirectionInvalid
	}
	if !services.IsValidSocialGiftEventType(eventType) {
		return errs.ErrSocialGiftEventTypeInvalid
	}
	if eventType == models.SOCIAL_GIFT_EVENT_OTHER && strings.TrimSpace(eventName) == "" {
		return errs.ErrSocialGiftEventNameRequired
	}
	return nil
}

func (a *SocialGiftsApi) ContactListHandler(c *core.WebContext) (any, *errs.Error) {
	if err := a.socialGifts.CheckPrivacy(c, c.GetCurrentUid(), c.GetHeader("X-Social-Gift-Token")); err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}
	var req models.SocialGiftContactListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}
	if req.Page < 1 {
		req.Page = 1
	}
	items, err := a.socialGifts.GetContactInfos(c, c.GetCurrentUid(), req.Keyword, req.Status)
	if err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}
	total := len(items)
	start := int((req.Page - 1) * req.Count)
	if start > total {
		start = total
	}
	end := start + int(req.Count)
	if end > total {
		end = total
	}
	return &models.SocialGiftContactPageResponse{Items: items[start:end], TotalCount: int64(total)}, nil
}

func (a *SocialGiftsApi) ContactGetHandler(c *core.WebContext) (any, *errs.Error) {
	if err := a.socialGifts.CheckPrivacy(c, c.GetCurrentUid(), c.GetHeader("X-Social-Gift-Token")); err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}
	id, err := utils.StringToInt64(c.Query("id"))
	if err != nil || id <= 0 {
		return nil, errs.ErrSocialGiftContactNotFound
	}
	info, err := a.socialGifts.GetContactInfo(c, c.GetCurrentUid(), id)
	if err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}
	return info, nil
}

func (a *SocialGiftsApi) ContactCreateHandler(c *core.WebContext) (any, *errs.Error) {
	if err := a.socialGifts.CheckPrivacy(c, c.GetCurrentUid(), c.GetHeader("X-Social-Gift-Token")); err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}
	var req models.SocialGiftContactCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}
	contact, err := a.socialGifts.CreateContact(c, c.GetCurrentUid(), &req)
	if err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}
	return &models.SocialGiftContactInfoResponse{Id: contact.ContactId, Name: contact.Name, Relationship: contact.Relationship, Comment: contact.Comment, Balances: []*models.SocialGiftBalanceResponse{}}, nil
}

func (a *SocialGiftsApi) ContactModifyHandler(c *core.WebContext) (any, *errs.Error) {
	if err := a.socialGifts.CheckPrivacy(c, c.GetCurrentUid(), c.GetHeader("X-Social-Gift-Token")); err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}
	var req models.SocialGiftContactModifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}
	if _, err := a.socialGifts.ModifyContact(c, c.GetCurrentUid(), &req); err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}
	info, err := a.socialGifts.GetContactInfo(c, c.GetCurrentUid(), req.Id)
	if err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}
	return info, nil
}

func (a *SocialGiftsApi) ContactDeleteHandler(c *core.WebContext) (any, *errs.Error) {
	if err := a.socialGifts.CheckPrivacy(c, c.GetCurrentUid(), c.GetHeader("X-Social-Gift-Token")); err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}
	var req models.SocialGiftContactDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}
	if err := a.socialGifts.DeleteContact(c, c.GetCurrentUid(), req.Id); err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}
	return true, nil
}

func (a *SocialGiftsApi) RecordListHandler(c *core.WebContext) (any, *errs.Error) {
	if err := a.socialGifts.CheckPrivacy(c, c.GetCurrentUid(), c.GetHeader("X-Social-Gift-Token")); err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}
	var req models.SocialGiftRecordListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}
	if req.Page < 1 {
		req.Page = 1
	}
	if req.Direction != "" && !services.IsValidSocialGiftDirection(req.Direction) {
		return nil, errs.ErrSocialGiftDirectionInvalid
	}
	if req.EventType != "" && !services.IsValidSocialGiftEventType(req.EventType) {
		return nil, errs.ErrSocialGiftEventTypeInvalid
	}
	items, err := a.socialGifts.GetRecordInfos(c, c.GetCurrentUid(), req.ContactId, req.Direction, req.EventType)
	if err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}
	total := len(items)
	start := int((req.Page - 1) * req.Count)
	if start > total {
		start = total
	}
	end := start + int(req.Count)
	if end > total {
		end = total
	}
	return &models.SocialGiftRecordPageResponse{Items: items[start:end], TotalCount: int64(total)}, nil
}

func (a *SocialGiftsApi) SummaryHandler(c *core.WebContext) (any, *errs.Error) {
	if err := a.socialGifts.CheckPrivacy(c, c.GetCurrentUid(), c.GetHeader("X-Social-Gift-Token")); err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}
	result, err := a.socialGifts.GetSummary(c, c.GetCurrentUid())
	if err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}
	return result, nil
}

func (a *SocialGiftsApi) RecordCreateHandler(c *core.WebContext) (any, *errs.Error) {
	if err := a.socialGifts.CheckPrivacy(c, c.GetCurrentUid(), c.GetHeader("X-Social-Gift-Token")); err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}
	var req models.SocialGiftRecordCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}
	if finalErr := validateSocialGiftMetadata(req.Direction, req.EventType, req.EventName); finalErr != nil {
		return nil, finalErr
	}
	if (req.ContactId > 0) == (req.NewContact != nil) {
		return nil, errs.ErrSocialGiftContactSelectionInvalid
	}
	uid := c.GetCurrentUid()
	user, err := a.users.GetUserById(c, uid)
	if err != nil {
		return nil, errs.ErrUserNotFound
	}
	clientTimezone, err := c.GetClientTimezone()
	if err != nil {
		return nil, errs.ErrClientTimezoneOffsetInvalid
	}
	transaction := newSocialGiftTransaction(uid, req.Direction, req.CategoryId, req.Time, req.UtcOffset, req.AccountId, req.Amount, req.Comment, c.ClientIP())
	if !user.CanEditTransactionByTransactionTime(transaction.TransactionTime, clientTimezone) {
		return nil, errs.ErrCannotCreateTransactionWithThisTransactionTime
	}
	if req.ClientSessionId != "" {
		if found, remark := a.GetSubmissionRemark(duplicatechecker.DUPLICATE_CHECKER_TYPE_NEW_SOCIAL_GIFT, uid, req.ClientSessionId); found {
			if recordId, parseErr := utils.StringToInt64(remark); parseErr == nil {
				record, getErr := a.socialGifts.GetRecord(c, uid, recordId)
				if getErr == nil {
					return a.findRecordInfo(c, uid, record.ContactId, record.RecordId)
				}
			}
		}
	}
	if req.Currency == "" {
		req.Currency = user.DefaultCurrency
	}
	record, _, err := a.socialGifts.CreateRecord(c, uid, &req, transaction)
	if err != nil {
		log.Errorf(c, "[social_gifts.RecordCreateHandler] failed to create record: %s", err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}
	a.SetSubmissionRemarkIfEnable(duplicatechecker.DUPLICATE_CHECKER_TYPE_NEW_SOCIAL_GIFT, uid, req.ClientSessionId, utils.Int64ToString(record.RecordId))
	return a.findRecordInfo(c, uid, record.ContactId, record.RecordId)
}

func (a *SocialGiftsApi) RecordModifyHandler(c *core.WebContext) (any, *errs.Error) {
	if err := a.socialGifts.CheckPrivacy(c, c.GetCurrentUid(), c.GetHeader("X-Social-Gift-Token")); err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}
	var req models.SocialGiftRecordModifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}
	if finalErr := validateSocialGiftMetadata(req.Direction, req.EventType, req.EventName); finalErr != nil {
		return nil, finalErr
	}
	uid := c.GetCurrentUid()
	record, err := a.socialGifts.GetRecord(c, uid, req.Id)
	if err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}
	oldTransaction, err := a.socialGifts.GetRecordTransaction(c, uid, record)
	if err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}
	user, err := a.users.GetUserById(c, uid)
	if err != nil {
		return nil, errs.ErrUserNotFound
	}
	clientTimezone, err := c.GetClientTimezone()
	if err != nil {
		return nil, errs.ErrClientTimezoneOffsetInvalid
	}
	newTransaction := newSocialGiftTransaction(uid, req.Direction, req.CategoryId, req.Time, req.UtcOffset, req.AccountId, req.Amount, req.Comment, c.ClientIP())
	if !user.CanEditTransactionByTransactionTime(oldTransaction.TransactionTime, clientTimezone) || !user.CanEditTransactionByTransactionTime(newTransaction.TransactionTime, clientTimezone) {
		return nil, errs.ErrCannotModifyTransactionWithThisTransactionTime
	}
	if req.Currency == "" {
		req.Currency = user.DefaultCurrency
	}
	record, err = a.socialGifts.ModifyRecord(c, uid, &req, newTransaction)
	if err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}
	return a.findRecordInfo(c, uid, record.ContactId, record.RecordId)
}

func (a *SocialGiftsApi) RecordDeleteHandler(c *core.WebContext) (any, *errs.Error) {
	if err := a.socialGifts.CheckPrivacy(c, c.GetCurrentUid(), c.GetHeader("X-Social-Gift-Token")); err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}
	var req models.SocialGiftRecordDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}
	uid := c.GetCurrentUid()
	record, err := a.socialGifts.GetRecord(c, uid, req.Id)
	if err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}
	transaction, err := a.socialGifts.GetRecordTransaction(c, uid, record)
	if err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}
	user, err := a.users.GetUserById(c, uid)
	if err != nil {
		return nil, errs.ErrUserNotFound
	}
	clientTimezone, err := c.GetClientTimezone()
	if err != nil {
		return nil, errs.ErrClientTimezoneOffsetInvalid
	}
	if !user.CanEditTransactionByTransactionTime(transaction.TransactionTime, clientTimezone) {
		return nil, errs.ErrCannotDeleteTransactionWithThisTransactionTime
	}
	if err = a.socialGifts.DeleteRecord(c, uid, req.Id); err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}
	return true, nil
}

func newSocialGiftTransaction(uid int64, direction string, categoryId int64, unixTime int64, utcOffset int16, accountId int64, amount int64, comment string, clientIp string) *models.Transaction {
	dbType := models.TRANSACTION_DB_TYPE_EXPENSE
	if direction == models.SOCIAL_GIFT_DIRECTION_RECEIVED {
		dbType = models.TRANSACTION_DB_TYPE_INCOME
	}
	return &models.Transaction{Uid: uid, Type: dbType, CategoryId: categoryId, TransactionTime: utils.GetMinTransactionTimeFromUnixTime(unixTime), TimezoneUtcOffset: utcOffset, AccountId: accountId, Amount: amount, Comment: strings.TrimSpace(comment), CreatedIp: clientIp}
}

func (a *SocialGiftsApi) findRecordInfo(c core.Context, uid, contactId, recordId int64) (any, *errs.Error) {
	items, err := a.socialGifts.GetRecordInfos(c, uid, contactId, "", "")
	if err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}
	for _, item := range items {
		if item.Id == recordId {
			return item, nil
		}
	}
	return nil, errs.ErrSocialGiftRecordNotFound
}
