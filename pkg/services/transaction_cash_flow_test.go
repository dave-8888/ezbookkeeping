package services

import (
	"testing"
	"time"

	"github.com/mayswind/financially_free/pkg/datastore"
	"github.com/mayswind/financially_free/pkg/models"
	"github.com/mayswind/financially_free/pkg/utils"
	"github.com/stretchr/testify/require"
)

func TestCashFlowLifecycleDoesNotChangeBalances(t *testing.T) {
	c := setupSocialGiftServiceTest(t)
	insertSocialGiftTestBasis(t, c, 1)
	require.NoError(t, datastore.Container.UserStore.SyncStructs(new(models.User)))
	_, err := Transactions.UserDB().NewSession(c).Insert(&models.User{Uid: 1, Username: "cashflow", DefaultCurrency: "CNY"})
	require.NoError(t, err)
	now := time.Now().Unix()
	for _, kind := range []models.TransactionDbType{models.TRANSACTION_DB_TYPE_EXPENSE, models.TRANSACTION_DB_TYPE_INCOME} {
		category := int64(112)
		if kind == models.TRANSACTION_DB_TYPE_INCOME {
			category = 122
		}
		row := &models.Transaction{Uid: 1, Type: kind, CategoryId: category, Amount: 3050, TransactionTime: utils.GetMinTransactionTimeFromUnixTime(now)}
		require.NoError(t, Transactions.CreateTransaction(c, row, nil, nil))
		require.Zero(t, row.AccountId)
		require.Equal(t, "CNY", row.Currency)
		row.Amount = 5000
		row.Currency = "USD"
		row.AccountId = 101 // stale clients cannot reattach it
		require.NoError(t, Transactions.ModifyTransaction(c, row, 0, nil, nil, nil, nil))
		saved, err := Transactions.GetTransactionByTransactionId(c, 1, row.TransactionId)
		require.NoError(t, err)
		require.Zero(t, saved.AccountId)
		require.Equal(t, "USD", saved.Currency)
		require.EqualValues(t, 5000, saved.Amount)
		require.True(t, saved.IsEditable(&models.User{Uid: 1, TransactionEditScope: models.TRANSACTION_EDIT_SCOPE_ALL}, time.UTC, nil, nil))
		require.Error(t, Transactions.DeleteTransaction(c, 2, row.TransactionId))
		require.NoError(t, Transactions.DeleteTransaction(c, 1, row.TransactionId))
	}
	account, err := Accounts.GetAccountByAccountId(c, 1, "101")
	require.NoError(t, err)
	require.Zero(t, account.Balance)
	invalid := &models.Transaction{Uid: 1, Type: models.TRANSACTION_DB_TYPE_EXPENSE, CategoryId: 112, Currency: "BAD", Amount: 1, TransactionTime: utils.GetMinTransactionTimeFromUnixTime(now)}
	require.Error(t, Transactions.CreateTransaction(c, invalid, nil, nil))
}

func TestDetachCashFlowPreservesCurrencyBalancesAndTransfers(t *testing.T) {
	c := setupSocialGiftServiceTest(t)
	insertSocialGiftTestBasis(t, c, 1)
	require.NoError(t, datastore.Container.UserDataStore.SyncStructs(new(models.TransactionTemplate), new(models.TransactionTag)))
	now := time.Now().Unix()
	for i, row := range []*models.Transaction{
		{TransactionId: 1, Uid: 1, Type: models.TRANSACTION_DB_TYPE_EXPENSE, CategoryId: 112, AccountId: 101, Amount: 300},
		{TransactionId: 2, Uid: 1, Type: models.TRANSACTION_DB_TYPE_EXPENSE, CategoryId: 112, AccountId: 102, Amount: 700},
		{TransactionId: 3, Uid: 1, Type: models.TRANSACTION_DB_TYPE_INCOME, CategoryId: 122, AccountId: 101, Amount: 900},
		{TransactionId: 4, Uid: 1, Type: models.TRANSACTION_DB_TYPE_TRANSFER_OUT, AccountId: 101, RelatedAccountId: 102, Amount: 600, RelatedAccountAmount: 100},
		{TransactionId: 5, Uid: 1, Type: models.TRANSACTION_DB_TYPE_EXPENSE, CategoryId: 112, AccountId: 101, Amount: 999, Deleted: true},
	} {
		row.TransactionTime = utils.GetMinTransactionTimeFromUnixTime(now) + int64(i)
		_, err := Transactions.UserDataDB(1).NewSession(c).Insert(row)
		require.NoError(t, err)
	}
	_, err := Transactions.UserDataDB(1).NewSession(c).Insert(&models.TransactionTemplate{TemplateId: 1, Uid: 1, Type: models.TRANSACTION_TYPE_EXPENSE, AccountId: 102, CategoryId: 112})
	require.NoError(t, err)
	require.NoError(t, Accounts.ModifyAccountBalance(c, 1, "101", 12345))
	for i := 0; i < 2; i++ {
		require.NoError(t, Transactions.DetachCashFlowAccounts(c))
	}
	var rows []*models.Transaction
	require.NoError(t, Transactions.UserDataDB(1).NewSession(c).OrderBy("transaction_id").Find(&rows))
	for _, row := range rows {
		if row.IsCashFlow() {
			require.Zero(t, row.AccountId)
			require.NotEmpty(t, row.Currency)
		} else {
			require.EqualValues(t, 101, row.AccountId)
			require.EqualValues(t, 102, row.RelatedAccountId)
		}
	}
	account, err := Accounts.GetAccountByAccountId(c, 1, "101")
	require.NoError(t, err)
	require.EqualValues(t, 12345, account.Balance)
	var template models.TransactionTemplate
	_, err = Transactions.UserDataDB(1).NewSession(c).ID(1).Get(&template)
	require.NoError(t, err)
	require.Zero(t, template.AccountId)
	require.Equal(t, "USD", template.Currency)
	income, expense, err := Transactions.GetAccountsTotalIncomeAndExpense(c, 1, now-10, now+10, nil, nil, time.UTC, false)
	require.NoError(t, err)
	require.EqualValues(t, 900, income["CNY"])
	require.EqualValues(t, 300, expense["CNY"])
	require.EqualValues(t, 700, expense["USD"])
	totals, err := Transactions.GetAccountsAndCategoriesTotalInflowAndOutflow(c, 1, now-10, now+10, nil, false, "", time.UTC, false)
	require.NoError(t, err)
	expenses := map[string]int64{}
	for _, row := range totals {
		if row.Type == models.TRANSACTION_DB_TYPE_EXPENSE {
			expenses[row.Currency] += row.Amount
		}
	}
	require.Equal(t, map[string]int64{"CNY": 300, "USD": 700}, expenses)
	// Editing/deleting migrated records must not refund their former account.
	rows[0].Amount = 500
	require.NoError(t, Transactions.ModifyTransaction(c, rows[0], 0, nil, nil, nil, nil))
	require.NoError(t, Transactions.DeleteTransaction(c, 1, rows[0].TransactionId))
	account, err = Accounts.GetAccountByAccountId(c, 1, "101")
	require.NoError(t, err)
	require.EqualValues(t, 12345, account.Balance)
}

func TestCashFlowTemplatesAndTransfers(t *testing.T) {
	c := setupSocialGiftServiceTest(t)
	insertSocialGiftTestBasis(t, c, 1)
	require.NoError(t, datastore.Container.UserDataStore.SyncStructs(new(models.TransactionTemplate), new(models.TransactionTag)))
	template := &models.TransactionTemplate{Uid: 1, Type: models.TRANSACTION_TYPE_EXPENSE, TemplateType: models.TRANSACTION_TEMPLATE_TYPE_NORMAL, CategoryId: 112, Currency: "CNY", Amount: 1234, Name: "Lunch"}
	require.NoError(t, TransactionTemplates.CreateTemplate(c, template))
	template.Currency = "USD"
	require.NoError(t, TransactionTemplates.ModifyTemplate(c, template))
	stored, err := TransactionTemplates.GetTemplateByTemplateId(c, 1, template.TemplateId)
	require.NoError(t, err)
	require.Equal(t, "USD", stored.Currency)
	require.Zero(t, stored.AccountId)
	for _, category := range []*models.TransactionCategory{
		{Uid: 1, CategoryId: 131, Type: models.CATEGORY_TYPE_TRANSFER, Name: "Transfers"},
		{Uid: 1, CategoryId: 132, ParentCategoryId: 131, Type: models.CATEGORY_TYPE_TRANSFER, Name: "Between accounts"},
	} {
		_, err := Transactions.UserDataDB(1).NewSession(c).Insert(category)
		require.NoError(t, err)
	}
	row := &models.Transaction{Uid: 1, Type: models.TRANSACTION_DB_TYPE_TRANSFER_OUT, CategoryId: 132, AccountId: 101, RelatedAccountId: 102, Amount: 700, RelatedAccountAmount: 100, TransactionTime: utils.GetMinTransactionTimeFromUnixTime(time.Now().Unix())}
	require.NoError(t, Transactions.CreateTransaction(c, row, nil, nil))
	first, err := Accounts.GetAccountByAccountId(c, 1, "101")
	require.NoError(t, err)
	require.EqualValues(t, -700, first.Balance)
	second, err := Accounts.GetAccountByAccountId(c, 1, "102")
	require.NoError(t, err)
	require.EqualValues(t, 100, second.Balance)
	require.NoError(t, Transactions.DeleteTransaction(c, 1, row.TransactionId))
	first, err = Accounts.GetAccountByAccountId(c, 1, "101")
	require.NoError(t, err)
	require.Zero(t, first.Balance)
}
