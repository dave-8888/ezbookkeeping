package services

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"xorm.io/xorm"

	"github.com/mayswind/financially_free/pkg/core"
	"github.com/mayswind/financially_free/pkg/datastore"
	"github.com/mayswind/financially_free/pkg/errs"
	"github.com/mayswind/financially_free/pkg/models"
	"github.com/mayswind/financially_free/pkg/utils"
	"github.com/mayswind/financially_free/pkg/uuid"
)

type SocialGiftService struct {
	ServiceUsingDB
	ServiceUsingUuid
	transactions *TransactionService
}

var SocialGifts = &SocialGiftService{
	ServiceUsingDB:   ServiceUsingDB{container: datastore.Container},
	ServiceUsingUuid: ServiceUsingUuid{container: uuid.Container},
	transactions:     Transactions,
}

type socialGiftRecordData struct {
	record      *models.SocialGiftRecord
	transaction *models.Transaction
	account     *models.Account
	category    *models.TransactionCategory
}

func IsValidSocialGiftDirection(direction string) bool {
	return direction == models.SOCIAL_GIFT_DIRECTION_SENT || direction == models.SOCIAL_GIFT_DIRECTION_RECEIVED
}

func IsValidSocialGiftEventType(eventType string) bool {
	switch eventType {
	case models.SOCIAL_GIFT_EVENT_WEDDING,
		models.SOCIAL_GIFT_EVENT_NEWBORN,
		models.SOCIAL_GIFT_EVENT_BIRTHDAY,
		models.SOCIAL_GIFT_EVENT_HOUSEWARMING,
		models.SOCIAL_GIFT_EVENT_EDUCATION,
		models.SOCIAL_GIFT_EVENT_FUNERAL,
		models.SOCIAL_GIFT_EVENT_OTHER:
		return true
	default:
		return false
	}
}

func socialGiftStatus(netAmount int64) string {
	if netAmount > 0 {
		return models.SOCIAL_GIFT_STATUS_RECEIVABLE
	} else if netAmount < 0 {
		return models.SOCIAL_GIFT_STATUS_PAYABLE
	}
	return models.SOCIAL_GIFT_STATUS_SETTLED
}

func (s *SocialGiftService) GetContact(c core.Context, uid int64, contactId int64) (*models.SocialGiftContact, error) {
	if uid <= 0 || contactId <= 0 {
		return nil, errs.ErrSocialGiftContactNotFound
	}
	contact := &models.SocialGiftContact{}
	has, err := s.UserDataDB(uid).NewSession(c).ID(contactId).Where("uid=? AND deleted=?", uid, false).Get(contact)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, errs.ErrSocialGiftContactNotFound
	}
	return contact, nil
}

func (s *SocialGiftService) CreateContact(c core.Context, uid int64, req *models.SocialGiftContactCreateRequest) (*models.SocialGiftContact, error) {
	contact := s.newContact(uid, req)
	_, err := s.UserDataDB(uid).NewSession(c).Insert(contact)
	if err != nil {
		return nil, err
	}
	return contact, nil
}

func (s *SocialGiftService) newContact(uid int64, req *models.SocialGiftContactCreateRequest) *models.SocialGiftContact {
	now := time.Now().Unix()
	return &models.SocialGiftContact{
		ContactId:       s.GenerateUuid(uuid.UUID_TYPE_SOCIAL_GIFT_CONTACT),
		Uid:             uid,
		Name:            strings.TrimSpace(req.Name),
		Relationship:    strings.TrimSpace(req.Relationship),
		Comment:         strings.TrimSpace(req.Comment),
		CreatedUnixTime: now,
		UpdatedUnixTime: now,
	}
}

func (s *SocialGiftService) ModifyContact(c core.Context, uid int64, req *models.SocialGiftContactModifyRequest) (*models.SocialGiftContact, error) {
	contact, err := s.GetContact(c, uid, req.Id)
	if err != nil {
		return nil, err
	}
	contact.Name = strings.TrimSpace(req.Name)
	contact.Relationship = strings.TrimSpace(req.Relationship)
	contact.Comment = strings.TrimSpace(req.Comment)
	contact.UpdatedUnixTime = time.Now().Unix()
	updated, err := s.UserDataDB(uid).NewSession(c).ID(contact.ContactId).Cols("name", "relationship", "comment", "updated_unix_time").Where("uid=? AND deleted=?", uid, false).Update(contact)
	if err != nil {
		return nil, err
	} else if updated < 1 {
		return nil, errs.ErrSocialGiftContactNotFound
	}
	return contact, nil
}

func (s *SocialGiftService) DeleteContact(c core.Context, uid int64, contactId int64) error {
	if _, err := s.GetContact(c, uid, contactId); err != nil {
		return err
	}
	sess := s.UserDataDB(uid).NewSession(c)
	inUse, err := sess.Where("uid=? AND deleted=? AND contact_id=?", uid, false, contactId).Exist(&models.SocialGiftRecord{})
	if err != nil {
		return err
	} else if inUse {
		return errs.ErrSocialGiftContactInUse
	}
	now := time.Now().Unix()
	updated, err := sess.ID(contactId).Cols("deleted", "deleted_unix_time", "updated_unix_time").Where("uid=? AND deleted=?", uid, false).Update(&models.SocialGiftContact{Deleted: true, DeletedUnixTime: now, UpdatedUnixTime: now})
	if err != nil {
		return err
	} else if updated < 1 {
		return errs.ErrSocialGiftContactNotFound
	}
	return nil
}

func (s *SocialGiftService) GetRecord(c core.Context, uid int64, recordId int64) (*models.SocialGiftRecord, error) {
	record := &models.SocialGiftRecord{}
	has, err := s.UserDataDB(uid).NewSession(c).ID(recordId).Where("uid=? AND deleted=?", uid, false).Get(record)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, errs.ErrSocialGiftRecordNotFound
	}
	return record, nil
}

// GetRecordTransaction returns linked accounting data or an in-memory view of a standalone gift.
// The standalone view is never persisted to the transactions table.
func (s *SocialGiftService) GetRecordTransaction(c core.Context, uid int64, record *models.SocialGiftRecord) (*models.Transaction, error) {
	if record.Uid != uid {
		return nil, errs.ErrSocialGiftRecordNotFound
	}
	if record.TransactionId > 0 {
		return s.transactions.GetTransactionByTransactionId(c, uid, record.TransactionId)
	}
	return standaloneSocialGiftTransaction(record), nil
}

func standaloneSocialGiftTransaction(record *models.SocialGiftRecord) *models.Transaction {
	dbType := models.TRANSACTION_DB_TYPE_EXPENSE
	if record.Direction == models.SOCIAL_GIFT_DIRECTION_RECEIVED {
		dbType = models.TRANSACTION_DB_TYPE_INCOME
	}
	return &models.Transaction{Uid: record.Uid, Type: dbType, Amount: record.Amount, TransactionTime: record.TransactionTime, TimezoneUtcOffset: record.TimezoneUtcOffset, AccountId: record.AccountId, CategoryId: record.CategoryId, Comment: record.Comment}
}

func (item *socialGiftRecordData) currency() string {
	if item.record.TransactionId > 0 && item.transaction != nil && item.transaction.Currency != "" {
		return item.transaction.Currency
	}
	if item.record.TransactionId > 0 && item.account != nil {
		return item.account.Currency
	}
	return item.record.Currency
}

func (s *SocialGiftService) prepareStandaloneRecord(c core.Context, record *models.SocialGiftRecord, transaction *models.Transaction, currency string) error {
	sess := s.UserDataDB(record.Uid).NewSession(c)
	defer sess.Close()
	if transaction.AccountId > 0 {
		account := &models.Account{}
		has, err := sess.ID(transaction.AccountId).Where("uid=? AND deleted=?", transaction.Uid, false).Get(account)
		if err == nil && !has {
			return errs.ErrSourceAccountNotFound
		}
		if err != nil {
			return err
		}
		if account.Hidden {
			return errs.ErrCannotAddTransactionToHiddenAccount
		}
		if account.Type == models.ACCOUNT_TYPE_MULTI_SUB_ACCOUNTS {
			return errs.ErrCannotAddTransactionToParentAccount
		}
		currency = account.Currency
	}
	if transaction.CategoryId > 0 {
		if err := s.transactions.isCategoryValid(sess, transaction); err != nil {
			return err
		}
	}
	record.TransactionId = -record.RecordId
	record.Direction = models.SOCIAL_GIFT_DIRECTION_SENT
	if transaction.Type == models.TRANSACTION_DB_TYPE_INCOME {
		record.Direction = models.SOCIAL_GIFT_DIRECTION_RECEIVED
	}
	record.Amount = transaction.Amount
	record.TransactionTime = transaction.TransactionTime
	record.TimezoneUtcOffset = transaction.TimezoneUtcOffset
	record.AccountId = transaction.AccountId
	record.CategoryId = transaction.CategoryId
	record.Currency = currency
	record.Comment = transaction.Comment
	return nil
}

func updateSocialGiftRecord(sess *xorm.Session, record *models.SocialGiftRecord) error {
	updated, err := sess.ID(record.RecordId).Cols("contact_id", "transaction_id", "event_type", "event_name", "direction", "amount", "transaction_time", "timezone_utc_offset", "account_id", "category_id", "currency", "comment", "deleted", "deleted_unix_time", "updated_unix_time").Where("uid=?", record.Uid).Update(record)
	if err != nil {
		return err
	}
	if updated < 1 {
		return errs.ErrSocialGiftRecordNotFound
	}
	return nil
}

func (s *SocialGiftService) CreateRecord(c core.Context, uid int64, req *models.SocialGiftRecordCreateRequest, transaction *models.Transaction) (*models.SocialGiftRecord, *models.SocialGiftContact, error) {
	var contact *models.SocialGiftContact
	var err error
	if req.ContactId > 0 {
		contact, err = s.GetContact(c, uid, req.ContactId)
		if err != nil {
			return nil, nil, err
		}
	} else if req.NewContact != nil {
		contact = s.newContact(uid, req.NewContact)
	} else {
		return nil, nil, errs.ErrSocialGiftContactSelectionInvalid
	}

	now := time.Now().Unix()
	record := &models.SocialGiftRecord{
		RecordId:        s.GenerateUuid(uuid.UUID_TYPE_SOCIAL_GIFT_RECORD),
		Uid:             uid,
		ContactId:       contact.ContactId,
		EventType:       req.EventType,
		EventName:       strings.TrimSpace(req.EventName),
		CreatedUnixTime: now,
		UpdatedUnixTime: now,
	}

	persist := func(sess *xorm.Session) error {
		if req.ContactId == 0 {
			if _, insertErr := sess.Insert(contact); insertErr != nil {
				return insertErr
			}
		}
		if transaction.TransactionId > 0 {
			record.TransactionId = transaction.TransactionId
		}
		_, insertErr := sess.Insert(record)
		return insertErr
	}
	if transaction.AccountId == 0 || transaction.CategoryId == 0 {
		if err = s.prepareStandaloneRecord(c, record, transaction, req.Currency); err != nil {
			return nil, nil, err
		}
		err = s.UserDataDB(uid).DoTransaction(c, persist)
	} else {
		err = s.transactions.CreateTransactionWithHook(c, transaction, nil, nil, persist)
	}
	if err != nil {
		return nil, nil, err
	}
	return record, contact, nil
}

func (s *SocialGiftService) ModifyRecord(c core.Context, uid int64, req *models.SocialGiftRecordModifyRequest, newTransaction *models.Transaction) (*models.SocialGiftRecord, error) {
	record, err := s.GetRecord(c, uid, req.Id)
	if err != nil {
		return nil, err
	}
	if _, err = s.GetContact(c, uid, req.ContactId); err != nil {
		return nil, err
	}
	oldLink := record.TransactionId
	record.ContactId = req.ContactId
	record.EventType = req.EventType
	record.EventName = strings.TrimSpace(req.EventName)
	record.UpdatedUnixTime = time.Now().Unix()
	if newTransaction.AccountId == 0 || newTransaction.CategoryId == 0 {
		if err = s.prepareStandaloneRecord(c, record, newTransaction, req.Currency); err != nil {
			return nil, err
		}
		persist := func(sess *xorm.Session) error { return updateSocialGiftRecord(sess, record) }
		if oldLink > 0 {
			// Deleting the old accounting entry also restores its account balance. The hook
			// keeps the gift active, atomically, after the linked-record deletion cascade.
			err = s.transactions.DeleteTransactionWithHook(c, uid, oldLink, persist)
		} else {
			err = s.UserDataDB(uid).DoTransaction(c, persist)
		}
		return record, err
	}
	if oldLink <= 0 {
		err = s.transactions.CreateTransactionWithHook(c, newTransaction, nil, nil, func(sess *xorm.Session) error {
			record.TransactionId = newTransaction.TransactionId
			return updateSocialGiftRecord(sess, record)
		})
		return record, err
	}
	oldTransaction, err := s.transactions.GetTransactionByTransactionId(c, uid, record.TransactionId)
	if err != nil {
		return nil, err
	}
	newTransaction.TransactionId = oldTransaction.TransactionId
	newTransaction.Uid = uid
	newTransaction.HideAmount = oldTransaction.HideAmount
	newTransaction.GeoLongitude = oldTransaction.GeoLongitude
	newTransaction.GeoLatitude = oldTransaction.GeoLatitude
	record.ContactId = req.ContactId
	record.EventType = req.EventType
	record.EventName = strings.TrimSpace(req.EventName)
	record.UpdatedUnixTime = time.Now().Unix()

	requestedType := models.TRANSACTION_DB_TYPE_EXPENSE
	if req.Direction == models.SOCIAL_GIFT_DIRECTION_RECEIVED {
		requestedType = models.TRANSACTION_DB_TYPE_INCOME
	}
	if oldTransaction.Type == requestedType {
		currentTagCount, countErr := s.UserDataDB(uid).NewSession(c).Where("uid=? AND deleted=? AND transaction_id=?", uid, false, oldTransaction.TransactionId).Count(&models.TransactionTagIndex{})
		if countErr != nil {
			return nil, countErr
		}
		err = s.transactions.ModifyTransactionWithHook(c, newTransaction, int(currentTagCount), nil, nil, nil, nil, func(sess *xorm.Session) error {
			updated, updateErr := sess.ID(record.RecordId).Cols("contact_id", "event_type", "event_name", "updated_unix_time").Where("uid=? AND deleted=?", uid, false).Update(record)
			if updateErr != nil {
				return updateErr
			} else if updated < 1 {
				return errs.ErrSocialGiftRecordNotFound
			}
			return nil
		})
		return record, err
	}

	// Changing direction replaces the linked transaction inside the deletion transaction.
	newTransaction.Type = requestedType
	newTransaction.TransactionId = s.transactions.GenerateUuid(uuid.UUID_TYPE_TRANSACTION)
	newTransaction.TransactionTime = utils.GetMinTransactionTimeFromUnixTime(utils.GetUnixTimeFromTransactionTime(newTransaction.TransactionTime))
	newTransaction.CreatedUnixTime = time.Now().Unix()
	newTransaction.UpdatedUnixTime = newTransaction.CreatedUnixTime
	if err = s.transactions.isAccountIdValid(newTransaction); err != nil {
		return nil, err
	}
	database := s.UserDataDB(uid)
	err = s.transactions.DeleteTransactionWithHook(c, uid, oldTransaction.TransactionId, func(sess *xorm.Session) error {
		if createErr := s.transactions.doCreateTransaction(c, database, sess, newTransaction, nil, nil, nil, nil); createErr != nil {
			return createErr
		}
		record.TransactionId = newTransaction.TransactionId
		record.Deleted = false
		record.DeletedUnixTime = 0
		updated, updateErr := sess.ID(record.RecordId).Cols("contact_id", "transaction_id", "event_type", "event_name", "deleted", "deleted_unix_time", "updated_unix_time").Where("uid=?", uid).Update(record)
		if updateErr != nil {
			return updateErr
		} else if updated < 1 {
			return errs.ErrSocialGiftRecordNotFound
		}
		return nil
	})
	return record, err
}

func (s *SocialGiftService) DeleteRecord(c core.Context, uid int64, recordId int64) error {
	record, err := s.GetRecord(c, uid, recordId)
	if err != nil {
		return err
	}
	if record.TransactionId <= 0 {
		record.Deleted = true
		record.DeletedUnixTime = time.Now().Unix()
		record.UpdatedUnixTime = record.DeletedUnixTime
		return s.UserDataDB(uid).DoTransaction(c, func(sess *xorm.Session) error { return updateSocialGiftRecord(sess, record) })
	}
	return s.transactions.DeleteTransaction(c, uid, record.TransactionId)
}

func (s *SocialGiftService) GetContactInfos(c core.Context, uid int64, keyword string, status string) ([]*models.SocialGiftContactInfoResponse, error) {
	var contacts []*models.SocialGiftContact
	if err := s.UserDataDB(uid).NewSession(c).Where("uid=? AND deleted=?", uid, false).Find(&contacts); err != nil {
		return nil, err
	}
	data, err := s.loadRecordData(c, uid, 0)
	if err != nil {
		return nil, err
	}
	byContact := make(map[int64][]*socialGiftRecordData)
	for _, item := range data {
		byContact[item.record.ContactId] = append(byContact[item.record.ContactId], item)
	}
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	result := make([]*models.SocialGiftContactInfoResponse, 0, len(contacts))
	for _, contact := range contacts {
		if keyword != "" && !strings.Contains(strings.ToLower(contact.Name+" "+contact.Relationship), keyword) {
			continue
		}
		info := buildSocialGiftContactInfo(contact, byContact[contact.ContactId])
		if status != "" && status != "all" && !contactInfoHasStatus(info, status) {
			continue
		}
		result = append(result, info)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].LastRecordTime != result[j].LastRecordTime {
			return result[i].LastRecordTime > result[j].LastRecordTime
		}
		return result[i].Name < result[j].Name
	})
	return result, nil
}

func (s *SocialGiftService) GetContactInfo(c core.Context, uid int64, contactId int64) (*models.SocialGiftContactInfoResponse, error) {
	contact, err := s.GetContact(c, uid, contactId)
	if err != nil {
		return nil, err
	}
	data, err := s.loadRecordData(c, uid, contactId)
	if err != nil {
		return nil, err
	}
	return buildSocialGiftContactInfo(contact, data), nil
}

func buildSocialGiftContactInfo(contact *models.SocialGiftContact, items []*socialGiftRecordData) *models.SocialGiftContactInfoResponse {
	info := &models.SocialGiftContactInfoResponse{Id: contact.ContactId, Name: contact.Name, Relationship: contact.Relationship, Comment: contact.Comment, Balances: make([]*models.SocialGiftBalanceResponse, 0)}
	balances := make(map[string]*models.SocialGiftBalanceResponse)
	for _, item := range items {
		balance := balances[item.currency()]
		if balance == nil {
			balance = &models.SocialGiftBalanceResponse{Currency: item.currency()}
			balances[item.currency()] = balance
		}
		if item.transaction.Type == models.TRANSACTION_DB_TYPE_EXPENSE {
			balance.SentAmount += item.transaction.Amount
		} else if item.transaction.Type == models.TRANSACTION_DB_TYPE_INCOME {
			balance.ReceivedAmount += item.transaction.Amount
		}
		info.RecordCount++
		timeValue := utils.GetUnixTimeFromTransactionTime(item.transaction.TransactionTime)
		if timeValue > info.LastRecordTime {
			info.LastRecordTime = timeValue
		}
	}
	for _, balance := range balances {
		balance.NetAmount = balance.SentAmount - balance.ReceivedAmount
		balance.Status = socialGiftStatus(balance.NetAmount)
		info.Balances = append(info.Balances, balance)
	}
	sort.Slice(info.Balances, func(i, j int) bool { return info.Balances[i].Currency < info.Balances[j].Currency })
	return info
}

func contactInfoHasStatus(info *models.SocialGiftContactInfoResponse, status string) bool {
	if len(info.Balances) == 0 {
		return status == models.SOCIAL_GIFT_STATUS_SETTLED
	}
	for _, balance := range info.Balances {
		if balance.Status == status {
			return true
		}
	}
	return false
}

func (s *SocialGiftService) GetRecordInfos(c core.Context, uid int64, contactId int64, direction string, eventType string) ([]*models.SocialGiftRecordInfoResponse, error) {
	if _, err := s.GetContact(c, uid, contactId); err != nil {
		return nil, err
	}
	data, err := s.loadRecordData(c, uid, contactId)
	if err != nil {
		return nil, err
	}
	result := make([]*models.SocialGiftRecordInfoResponse, 0, len(data))
	for _, item := range data {
		itemDirection := models.SOCIAL_GIFT_DIRECTION_SENT
		if item.transaction.Type == models.TRANSACTION_DB_TYPE_INCOME {
			itemDirection = models.SOCIAL_GIFT_DIRECTION_RECEIVED
		}
		if direction != "" && direction != itemDirection {
			continue
		}
		if eventType != "" && eventType != item.record.EventType {
			continue
		}
		transactionResponse := item.transaction.ToTransactionInfoResponse([]int64{}, true)
		if item.account != nil {
			transactionResponse.SourceAccount = item.account.ToAccountInfoResponse()
		}
		if item.category != nil {
			transactionResponse.Category = item.category.ToTransactionCategoryInfoResponse()
		}
		result = append(result, &models.SocialGiftRecordInfoResponse{Id: item.record.RecordId, ContactId: item.record.ContactId, Direction: itemDirection, EventType: item.record.EventType, EventName: item.record.EventName, Currency: item.currency(), Transaction: transactionResponse})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Transaction.TimeSequenceId > result[j].Transaction.TimeSequenceId
	})
	return result, nil
}

func (s *SocialGiftService) GetSummary(c core.Context, uid int64) (*models.SocialGiftSummaryResponse, error) {
	contacts, err := s.GetContactInfos(c, uid, "", "")
	if err != nil {
		return nil, err
	}
	response := &models.SocialGiftSummaryResponse{Balances: make([]*models.SocialGiftBalanceResponse, 0)}
	balances := make(map[string]*models.SocialGiftBalanceResponse)
	for _, contact := range contacts {
		hasReceivable, hasPayable := false, false
		for _, item := range contact.Balances {
			balance := balances[item.Currency]
			if balance == nil {
				balance = &models.SocialGiftBalanceResponse{Currency: item.Currency}
				balances[item.Currency] = balance
			}
			balance.SentAmount += item.SentAmount
			balance.ReceivedAmount += item.ReceivedAmount
			hasReceivable = hasReceivable || item.Status == models.SOCIAL_GIFT_STATUS_RECEIVABLE
			hasPayable = hasPayable || item.Status == models.SOCIAL_GIFT_STATUS_PAYABLE
		}
		if hasReceivable {
			response.ReceivableContactCount++
		}
		if hasPayable {
			response.PayableContactCount++
		}
	}
	for _, balance := range balances {
		balance.NetAmount = balance.SentAmount - balance.ReceivedAmount
		balance.Status = socialGiftStatus(balance.NetAmount)
		response.Balances = append(response.Balances, balance)
	}
	sort.Slice(response.Balances, func(i, j int) bool { return response.Balances[i].Currency < response.Balances[j].Currency })
	return response, nil
}

func (s *SocialGiftService) loadRecordData(c core.Context, uid int64, contactId int64) ([]*socialGiftRecordData, error) {
	condition := "uid=? AND deleted=?"
	args := []any{uid, false}
	if contactId > 0 {
		condition += " AND contact_id=?"
		args = append(args, contactId)
	}
	var records []*models.SocialGiftRecord
	if err := s.UserDataDB(uid).NewSession(c).Where(condition, args...).Find(&records); err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return []*socialGiftRecordData{}, nil
	}
	transactionIds := make([]int64, 0, len(records))
	for _, record := range records {
		if record.TransactionId > 0 {
			transactionIds = append(transactionIds, record.TransactionId)
		}
	}
	var transactions []*models.Transaction
	if len(transactionIds) > 0 {
		if err := s.UserDataDB(uid).NewSession(c).Where("uid=? AND deleted=?", uid, false).In("transaction_id", transactionIds).Find(&transactions); err != nil {
			return nil, err
		}
	}
	transactionMap := make(map[int64]*models.Transaction)
	accountIds := make([]int64, 0, len(transactions))
	categoryIds := make([]int64, 0, len(transactions))
	for _, transaction := range transactions {
		if transaction.Type != models.TRANSACTION_DB_TYPE_EXPENSE && transaction.Type != models.TRANSACTION_DB_TYPE_INCOME {
			continue
		}
		transactionMap[transaction.TransactionId] = transaction
		accountIds = append(accountIds, transaction.AccountId)
		categoryIds = append(categoryIds, transaction.CategoryId)
	}
	for _, record := range records {
		if record.TransactionId <= 0 {
			accountIds = append(accountIds, record.AccountId)
			categoryIds = append(categoryIds, record.CategoryId)
		}
	}
	var accounts []*models.Account
	if len(accountIds) > 0 {
		if err := s.UserDataDB(uid).NewSession(c).Where("uid=? AND deleted=?", uid, false).In("account_id", accountIds).Find(&accounts); err != nil {
			return nil, err
		}
	}
	accountMap := make(map[int64]*models.Account)
	for _, account := range accounts {
		accountMap[account.AccountId] = account
	}
	var categories []*models.TransactionCategory
	if len(categoryIds) > 0 {
		if err := s.UserDataDB(uid).NewSession(c).Where("uid=? AND deleted=?", uid, false).In("category_id", categoryIds).Find(&categories); err != nil {
			return nil, err
		}
	}
	categoryMap := make(map[int64]*models.TransactionCategory)
	for _, category := range categories {
		categoryMap[category.CategoryId] = category
	}
	result := make([]*socialGiftRecordData, 0, len(records))
	for _, record := range records {
		transaction := transactionMap[record.TransactionId]
		if record.TransactionId <= 0 {
			transaction = standaloneSocialGiftTransaction(record)
		} else if transaction == nil || (!transaction.IsCashFlow() && accountMap[transaction.AccountId] == nil) {
			continue
		}
		result = append(result, &socialGiftRecordData{record: record, transaction: transaction, account: accountMap[transaction.AccountId], category: categoryMap[transaction.CategoryId]})
	}
	return result, nil
}

func (s *SocialGiftService) GetTotalContactCountByUid(c core.Context, uid int64) (int64, error) {
	return s.UserDataDB(uid).NewSession(c).Where("uid=? AND deleted=?", uid, false).Count(&models.SocialGiftContact{})
}

func (s *SocialGiftService) GetTotalRecordCountByUid(c core.Context, uid int64) (int64, error) {
	return s.UserDataDB(uid).NewSession(c).Where("uid=? AND deleted=?", uid, false).Count(&models.SocialGiftRecord{})
}

// AttachExportMetadata adds social gift fields only to the supplied transactions.
func (s *SocialGiftService) AttachExportMetadata(c core.Context, uid int64, transactions []*models.Transaction) error {
	if len(transactions) == 0 {
		return nil
	}
	transactionMap := make(map[int64]*models.Transaction, len(transactions))
	transactionIds := make([]int64, 0, len(transactions))
	for _, transaction := range transactions {
		transactionMap[transaction.TransactionId] = transaction
		transactionIds = append(transactionIds, transaction.TransactionId)
	}
	var records []*models.SocialGiftRecord
	if err := s.UserDataDB(uid).NewSession(c).Where("uid=? AND deleted=?", uid, false).In("transaction_id", transactionIds).Find(&records); err != nil {
		return err
	}
	if len(records) == 0 {
		return nil
	}
	contactIds := make([]int64, 0, len(records))
	for _, record := range records {
		contactIds = append(contactIds, record.ContactId)
	}
	var contacts []*models.SocialGiftContact
	if err := s.UserDataDB(uid).NewSession(c).Where("uid=? AND deleted=?", uid, false).In("contact_id", contactIds).Find(&contacts); err != nil {
		return err
	}
	contactMap := make(map[int64]*models.SocialGiftContact, len(contacts))
	for _, contact := range contacts {
		contactMap[contact.ContactId] = contact
	}
	for _, record := range records {
		transaction, contact := transactionMap[record.TransactionId], contactMap[record.ContactId]
		if transaction == nil || contact == nil {
			continue
		}
		transaction.SocialGiftContactKey = strconv.FormatInt(contact.ContactId, 10)
		transaction.SocialGiftContact = contact.Name
		transaction.SocialGiftRelationship = contact.Relationship
		transaction.SocialGiftContactNote = contact.Comment
		transaction.SocialGiftEventType = record.EventType
		transaction.SocialGiftEventName = record.EventName
	}
	return nil
}

// BuildImportHook validates imported social gift metadata and builds an atomic persistence hook.
func (s *SocialGiftService) BuildImportHook(uid int64, requests []*models.TransactionCreateRequest, transactions []*models.Transaction) (TransactionSessionHook, error) {
	if len(requests) != len(transactions) {
		return nil, errs.ErrOperationFailed
	}
	now := time.Now().Unix()
	contactsByKey := make(map[string]*models.SocialGiftContact)
	contacts := make([]*models.SocialGiftContact, 0)
	records := make([]*models.SocialGiftRecord, 0)
	recordTransactions := make([]*models.Transaction, 0)
	for index, request := range requests {
		metadata := request.SocialGift
		if metadata == nil {
			continue
		}
		metadata.ContactName = strings.TrimSpace(metadata.ContactName)
		metadata.Relationship = strings.TrimSpace(metadata.Relationship)
		metadata.ContactNote = strings.TrimSpace(metadata.ContactNote)
		metadata.EventName = strings.TrimSpace(metadata.EventName)
		if metadata.ContactName == "" {
			return nil, errs.ErrSocialGiftContactSelectionInvalid
		}
		if !IsValidSocialGiftEventType(metadata.EventType) {
			return nil, errs.ErrSocialGiftEventTypeInvalid
		}
		if metadata.EventType == models.SOCIAL_GIFT_EVENT_OTHER && metadata.EventName == "" {
			return nil, errs.ErrSocialGiftEventNameRequired
		}
		transaction := transactions[index]
		if transaction.Amount <= 0 || (transaction.Type != models.TRANSACTION_DB_TYPE_EXPENSE && transaction.Type != models.TRANSACTION_DB_TYPE_INCOME) {
			return nil, errs.ErrSocialGiftDirectionInvalid
		}
		key := strings.TrimSpace(metadata.ContactKey)
		generatedKey := key == ""
		if key == "" {
			key = "normalized:" + strings.ToLower(strings.Join(strings.Fields(metadata.ContactName), " ")) + "\x00" + strings.ToLower(strings.Join(strings.Fields(metadata.Relationship), " "))
		}
		contact := contactsByKey[key]
		if contact == nil {
			contact = &models.SocialGiftContact{ContactId: s.GenerateUuid(uuid.UUID_TYPE_SOCIAL_GIFT_CONTACT), Uid: uid, Name: metadata.ContactName, Relationship: metadata.Relationship, Comment: metadata.ContactNote, CreatedUnixTime: now, UpdatedUnixTime: now}
			contactsByKey[key] = contact
			contacts = append(contacts, contact)
		} else {
			if !generatedKey && (contact.Name != metadata.ContactName || contact.Relationship != metadata.Relationship || contact.Comment != metadata.ContactNote) {
				return nil, errs.ErrSocialGiftImportContactConflict
			}
			if generatedKey && contact.Comment != "" && metadata.ContactNote != "" && contact.Comment != metadata.ContactNote {
				return nil, errs.ErrSocialGiftImportContactConflict
			}
			if contact.Comment == "" {
				contact.Comment = metadata.ContactNote
			}
		}
		records = append(records, &models.SocialGiftRecord{RecordId: s.GenerateUuid(uuid.UUID_TYPE_SOCIAL_GIFT_RECORD), Uid: uid, ContactId: contact.ContactId, EventType: metadata.EventType, EventName: metadata.EventName, CreatedUnixTime: now, UpdatedUnixTime: now})
		recordTransactions = append(recordTransactions, transaction)
	}
	if len(records) == 0 {
		return nil, nil
	}
	return func(sess *xorm.Session) error {
		for _, contact := range contacts {
			if _, err := sess.Insert(contact); err != nil {
				return err
			}
		}
		for index, record := range records {
			record.TransactionId = recordTransactions[index].TransactionId
			if _, err := sess.Insert(record); err != nil {
				return err
			}
		}
		return nil
	}, nil
}

func (s *SocialGiftService) DeleteAllContacts(c core.Context, uid int64) error {
	now := time.Now().Unix()
	_, err := s.UserDataDB(uid).NewSession(c).Cols("deleted", "deleted_unix_time", "updated_unix_time").Where("uid=? AND deleted=?", uid, false).Update(&models.SocialGiftContact{Deleted: true, DeletedUnixTime: now, UpdatedUnixTime: now})
	return err
}
