package services

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mayswind/financially_free/pkg/core"
	"github.com/mayswind/financially_free/pkg/datastore"
	"github.com/mayswind/financially_free/pkg/errs"
	"github.com/mayswind/financially_free/pkg/models"
	"github.com/mayswind/financially_free/pkg/settings"
	"github.com/mayswind/financially_free/pkg/utils"
	"github.com/mayswind/financially_free/pkg/uuid"
)

func setupSocialGiftServiceTest(t *testing.T) core.Context {
	t.Helper()
	config := &settings.Config{
		DatabaseConfig:    &settings.DatabaseConfig{DatabaseType: settings.Sqlite3DbType, DatabasePath: filepath.Join(t.TempDir(), "social-gifts.db")},
		UuidGeneratorType: settings.InternalUuidGeneratorType,
	}
	require.NoError(t, datastore.InitializeDataStore(config))
	require.NoError(t, uuid.InitializeUuidGenerator(config))
	require.NoError(t, datastore.Container.UserDataStore.SyncStructs(
		new(models.Account), new(models.TransactionCategory), new(models.Transaction), new(models.TransactionTagIndex),
		new(models.TransactionPictureInfo), new(models.SocialGiftContact), new(models.SocialGiftRecord),
	))
	return core.NewNullContext()
}

func insertSocialGiftTestBasis(t *testing.T, context core.Context, uid int64) {
	t.Helper()
	now := time.Now().Unix()
	items := []any{
		&models.Account{AccountId: uid*100 + 1, Uid: uid, Category: models.ACCOUNT_CATEGORY_CASH, Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT, Name: "CNY Cash", Currency: "CNY", CreatedUnixTime: now, UpdatedUnixTime: now},
		&models.Account{AccountId: uid*100 + 2, Uid: uid, Category: models.ACCOUNT_CATEGORY_CASH, Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT, Name: "USD Cash", Currency: "USD", CreatedUnixTime: now, UpdatedUnixTime: now},
		&models.TransactionCategory{CategoryId: uid*100 + 11, Uid: uid, Type: models.CATEGORY_TYPE_EXPENSE, Name: "Expense", CreatedUnixTime: now, UpdatedUnixTime: now},
		&models.TransactionCategory{CategoryId: uid*100 + 12, Uid: uid, Type: models.CATEGORY_TYPE_EXPENSE, ParentCategoryId: uid*100 + 11, Name: "Gift Expense", CreatedUnixTime: now, UpdatedUnixTime: now},
		&models.TransactionCategory{CategoryId: uid*100 + 21, Uid: uid, Type: models.CATEGORY_TYPE_INCOME, Name: "Income", CreatedUnixTime: now, UpdatedUnixTime: now},
		&models.TransactionCategory{CategoryId: uid*100 + 22, Uid: uid, Type: models.CATEGORY_TYPE_INCOME, ParentCategoryId: uid*100 + 21, Name: "Gift Income", CreatedUnixTime: now, UpdatedUnixTime: now},
	}
	for _, item := range items {
		_, err := SocialGifts.UserDataDB(uid).NewSession(context).Insert(item)
		require.NoError(t, err)
	}
}

func createSocialGiftTestRecord(t *testing.T, context core.Context, uid int64, contactId int64, newContact *models.SocialGiftContactCreateRequest, direction string, accountId, categoryId, amount int64) (*models.SocialGiftRecord, *models.SocialGiftContact) {
	t.Helper()
	dbType := models.TRANSACTION_DB_TYPE_EXPENSE
	if direction == models.SOCIAL_GIFT_DIRECTION_RECEIVED {
		dbType = models.TRANSACTION_DB_TYPE_INCOME
	}
	req := &models.SocialGiftRecordCreateRequest{Currency: "CNY", ContactId: contactId, NewContact: newContact, Direction: direction, EventType: models.SOCIAL_GIFT_EVENT_WEDDING, AccountId: accountId, CategoryId: categoryId, Amount: amount}
	transaction := &models.Transaction{Uid: uid, Type: dbType, CategoryId: categoryId, TransactionTime: utils.GetMinTransactionTimeFromUnixTime(time.Now().Unix()), AccountId: accountId, Amount: amount}
	record, contact, err := SocialGifts.CreateRecord(context, uid, req, transaction)
	require.NoError(t, err)
	return record, contact
}

func TestSocialGiftServiceBalancesIsolationAndLifecycle(t *testing.T) {
	context := setupSocialGiftServiceTest(t)
	insertSocialGiftTestBasis(t, context, 1)
	insertSocialGiftTestBasis(t, context, 2)

	sent, contact := createSocialGiftTestRecord(t, context, 1, 0, &models.SocialGiftContactCreateRequest{Name: "Alex", Relationship: "Friend"}, models.SOCIAL_GIFT_DIRECTION_SENT, 101, 112, 10000)
	createSocialGiftTestRecord(t, context, 1, contact.ContactId, nil, models.SOCIAL_GIFT_DIRECTION_RECEIVED, 101, 122, 3000)
	createSocialGiftTestRecord(t, context, 1, contact.ContactId, nil, models.SOCIAL_GIFT_DIRECTION_RECEIVED, 102, 122, 2000)
	_, otherContact := createSocialGiftTestRecord(t, context, 2, 0, &models.SocialGiftContactCreateRequest{Name: "Alex", Relationship: "Friend"}, models.SOCIAL_GIFT_DIRECTION_RECEIVED, 201, 222, 500)
	assert.NotEqual(t, contact.ContactId, otherContact.ContactId)

	info, err := SocialGifts.GetContactInfo(context, 1, contact.ContactId)
	require.NoError(t, err)
	require.Len(t, info.Balances, 2)
	assert.Equal(t, int64(7000), info.Balances[0].NetAmount)
	assert.Equal(t, models.SOCIAL_GIFT_STATUS_RECEIVABLE, info.Balances[0].Status)
	assert.Equal(t, int64(-2000), info.Balances[1].NetAmount)
	assert.Equal(t, models.SOCIAL_GIFT_STATUS_PAYABLE, info.Balances[1].Status)

	receivable, err := SocialGifts.GetContactInfos(context, 1, "alex", models.SOCIAL_GIFT_STATUS_RECEIVABLE)
	require.NoError(t, err)
	assert.Len(t, receivable, 1)
	payable, err := SocialGifts.GetContactInfos(context, 1, "friend", models.SOCIAL_GIFT_STATUS_PAYABLE)
	require.NoError(t, err)
	assert.Len(t, payable, 1)
	otherUser, err := SocialGifts.GetContactInfos(context, 2, "", "")
	require.NoError(t, err)
	assert.Len(t, otherUser, 1)

	err = SocialGifts.DeleteContact(context, 1, contact.ContactId)
	assert.ErrorIs(t, err, errs.ErrSocialGiftContactInUse)
	require.NoError(t, Transactions.DeleteTransaction(context, 1, sent.TransactionId))
	_, err = SocialGifts.GetRecord(context, 1, sent.RecordId)
	assert.ErrorIs(t, err, errs.ErrSocialGiftRecordNotFound)
}

func TestSocialGiftServiceCreateRollbackAndImportedGrouping(t *testing.T) {
	context := setupSocialGiftServiceTest(t)
	insertSocialGiftTestBasis(t, context, 3)

	badReq := &models.SocialGiftRecordCreateRequest{NewContact: &models.SocialGiftContactCreateRequest{Name: "Rollback"}, Direction: models.SOCIAL_GIFT_DIRECTION_SENT, EventType: models.SOCIAL_GIFT_EVENT_WEDDING, CategoryId: 399, AccountId: 301, Amount: 100}
	badTransaction := &models.Transaction{Uid: 3, Type: models.TRANSACTION_DB_TYPE_EXPENSE, CategoryId: 399, TransactionTime: utils.GetMinTransactionTimeFromUnixTime(time.Now().Unix()), AccountId: 301, Amount: 100}
	_, _, err := SocialGifts.CreateRecord(context, 3, badReq, badTransaction)
	assert.ErrorIs(t, err, errs.ErrTransactionCategoryNotFound)
	count, err := SocialGifts.GetTotalContactCountByUid(context, 3)
	require.NoError(t, err)
	assert.Zero(t, count)

	requests := []*models.TransactionCreateRequest{
		{SocialGift: &models.SocialGiftImportMetadata{ContactKey: "same", ContactName: "Taylor", Relationship: "Colleague", EventType: models.SOCIAL_GIFT_EVENT_WEDDING}},
		{SocialGift: &models.SocialGiftImportMetadata{ContactKey: "same", ContactName: "Taylor", Relationship: "Colleague", EventType: models.SOCIAL_GIFT_EVENT_BIRTHDAY}},
	}
	transactions := []*models.Transaction{
		{Uid: 3, Type: models.TRANSACTION_DB_TYPE_EXPENSE, CategoryId: 312, TransactionTime: utils.GetMinTransactionTimeFromUnixTime(time.Now().Unix()), AccountId: 301, Amount: 800},
		{Uid: 3, Type: models.TRANSACTION_DB_TYPE_INCOME, CategoryId: 322, TransactionTime: utils.GetMinTransactionTimeFromUnixTime(time.Now().Add(time.Second).Unix()), AccountId: 301, Amount: 300},
	}
	hook, err := SocialGifts.BuildImportHook(3, requests, transactions)
	require.NoError(t, err)
	require.NoError(t, Transactions.BatchCreateTransactionsWithHook(context, 3, transactions, map[int][]int64{}, nil, hook))
	count, err = SocialGifts.GetTotalContactCountByUid(context, 3)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
	recordCount, err := SocialGifts.GetTotalRecordCountByUid(context, 3)
	require.NoError(t, err)
	assert.Equal(t, int64(2), recordCount)
}

func TestSocialGiftStandaloneRecordsAndAccountingTransitions(t *testing.T) {
	context := setupSocialGiftServiceTest(t)
	insertSocialGiftTestBasis(t, context, 1)
	insertSocialGiftTestBasis(t, context, 2)
	record, contact := createSocialGiftTestRecord(t, context, 1, 0, &models.SocialGiftContactCreateRequest{Name: "Alex"}, models.SOCIAL_GIFT_DIRECTION_SENT, 0, 0, 50000)
	// Multiple standalone gifts must coexist despite the unique transaction-link index.
	second, _ := createSocialGiftTestRecord(t, context, 1, contact.ContactId, nil, models.SOCIAL_GIFT_DIRECTION_RECEIVED, 0, 122, 20000)
	third, _ := createSocialGiftTestRecord(t, context, 1, contact.ContactId, nil, models.SOCIAL_GIFT_DIRECTION_SENT, 102, 0, 1000)
	assert.Less(t, record.TransactionId, int64(0))
	assert.NotEqual(t, record.TransactionId, second.TransactionId)
	assertAccounting := func(count, balance int64) {
		t.Helper()
		actualCount, err := Transactions.GetTotalTransactionCountByUid(context, 1)
		require.NoError(t, err)
		assert.Equal(t, count, actualCount)
		account := &models.Account{}
		found, err := SocialGifts.UserDataDB(1).NewSession(context).ID(101).Get(account)
		require.NoError(t, err)
		require.True(t, found)
		assert.Equal(t, balance, account.Balance)
	}
	assertAccounting(0, 0)
	info, err := SocialGifts.GetContactInfo(context, 1, contact.ContactId)
	require.NoError(t, err)
	assert.Equal(t, int64(3), info.RecordCount)
	require.Len(t, info.Balances, 2)
	assert.Equal(t, "CNY", info.Balances[0].Currency)
	assert.Equal(t, int64(30000), info.Balances[0].NetAmount)
	assert.Equal(t, "USD", info.Balances[1].Currency)
	items, err := SocialGifts.GetRecordInfos(context, 1, contact.ContactId, "received", "")
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Zero(t, items[0].Transaction.Id)
	assert.Zero(t, items[0].Transaction.SourceAccountId)
	assert.Equal(t, int64(122), items[0].Transaction.CategoryId)
	_, err = SocialGifts.GetRecord(context, 2, record.RecordId)
	assert.ErrorIs(t, err, errs.ErrSocialGiftRecordNotFound)
	require.NoError(t, SocialGifts.DeleteRecord(context, 1, third.RecordId))
	require.NoError(t, SocialGifts.DeleteRecord(context, 1, second.RecordId))
	assertAccounting(0, 0)

	modify := func(account, category, amount int64, direction string) error {
		dbType := models.TRANSACTION_DB_TYPE_EXPENSE
		if direction == models.SOCIAL_GIFT_DIRECTION_RECEIVED {
			dbType = models.TRANSACTION_DB_TYPE_INCOME
		}
		req := &models.SocialGiftRecordModifyRequest{Id: record.RecordId, ContactId: contact.ContactId, Direction: direction, EventType: models.SOCIAL_GIFT_EVENT_BIRTHDAY, Currency: "CNY"}
		transaction := &models.Transaction{Uid: 1, Type: dbType, AccountId: account, CategoryId: category, Amount: amount, TransactionTime: utils.GetMinTransactionTimeFromUnixTime(time.Now().Unix()), TimezoneUtcOffset: 480, Comment: "updated"}
		_, err := SocialGifts.ModifyRecord(context, 1, req, transaction)
		return err
	}
	require.NoError(t, modify(0, 0, 60000, models.SOCIAL_GIFT_DIRECTION_SENT))
	assertAccounting(0, 0)
	// Invalid link creation rolls back without losing the standalone record.
	require.Error(t, modify(101, 999, 60000, models.SOCIAL_GIFT_DIRECTION_SENT))
	assertAccounting(0, 0)
	saved, err := SocialGifts.GetRecord(context, 1, record.RecordId)
	require.NoError(t, err)
	assert.Equal(t, int64(60000), saved.Amount)
	require.NoError(t, modify(101, 112, 60000, models.SOCIAL_GIFT_DIRECTION_SENT))
	assertAccounting(1, 0)
	require.NoError(t, modify(101, 112, 70000, models.SOCIAL_GIFT_DIRECTION_SENT))
	assertAccounting(1, 0)
	require.NoError(t, modify(101, 122, 80000, models.SOCIAL_GIFT_DIRECTION_RECEIVED))
	assertAccounting(1, 0)
	// Clearing a field atomically removes the accounting entry without changing balance.
	require.NoError(t, modify(101, 0, 90000, models.SOCIAL_GIFT_DIRECTION_RECEIVED))
	assertAccounting(0, 0)
	saved, err = SocialGifts.GetRecord(context, 1, record.RecordId)
	require.NoError(t, err)
	assert.False(t, saved.Deleted)
	assert.Less(t, saved.TransactionId, int64(0))
	transaction, err := SocialGifts.GetRecordTransaction(context, 1, saved)
	require.NoError(t, err)
	assert.Equal(t, int64(90000), transaction.Amount)
	assert.Equal(t, int16(480), transaction.TimezoneUtcOffset)
	assert.Equal(t, "updated", transaction.Comment)
	require.NoError(t, modify(0, 0, 90000, models.SOCIAL_GIFT_DIRECTION_RECEIVED))
	require.NoError(t, SocialGifts.DeleteRecord(context, 1, record.RecordId))
	assertAccounting(0, 0)
	_, err = SocialGifts.GetRecord(context, 1, record.RecordId)
	assert.ErrorIs(t, err, errs.ErrSocialGiftRecordNotFound)
	require.NoError(t, SocialGifts.DeleteContact(context, 1, contact.ContactId))
}

func TestSocialGiftStandaloneRejectsOtherUsersSelections(t *testing.T) {
	context := setupSocialGiftServiceTest(t)
	insertSocialGiftTestBasis(t, context, 2)
	for _, selection := range [][2]int64{{201, 0}, {0, 222}} {
		req := &models.SocialGiftRecordCreateRequest{NewContact: &models.SocialGiftContactCreateRequest{Name: "Invalid"}, Direction: models.SOCIAL_GIFT_DIRECTION_SENT, EventType: models.SOCIAL_GIFT_EVENT_WEDDING, Currency: "CNY"}
		transaction := &models.Transaction{Uid: 1, Type: models.TRANSACTION_DB_TYPE_EXPENSE, AccountId: selection[0], CategoryId: selection[1], Amount: 100, TransactionTime: utils.GetMinTransactionTimeFromUnixTime(time.Now().Unix())}
		_, _, err := SocialGifts.CreateRecord(context, 1, req, transaction)
		require.Error(t, err)
	}
	count, err := SocialGifts.GetTotalContactCountByUid(context, 1)
	require.NoError(t, err)
	assert.Zero(t, count)
}
