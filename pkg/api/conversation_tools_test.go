package api

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/mayswind/financially_free/pkg/datastore"
	"github.com/mayswind/financially_free/pkg/models"
	"github.com/mayswind/financially_free/pkg/services"
	"github.com/mayswind/financially_free/pkg/settings"
	"github.com/mayswind/financially_free/pkg/uuid"
	"github.com/mayswind/financially_free/pkg/validators"
	"github.com/stretchr/testify/require"
)

func TestConversationToolsUseAuthenticatedAccountsAndCheckedBalanceWrite(t *testing.T) {
	config := &settings.Config{DatabaseConfig: &settings.DatabaseConfig{DatabaseType: settings.Sqlite3DbType, DatabasePath: filepath.Join(t.TempDir(), "tools.db")}}
	require.NoError(t, datastore.InitializeDataStore(config))
	require.NoError(t, datastore.Container.UserDataStore.SyncStructs(new(models.Account), new(models.Transaction)))
	c := newAssetPlanTestWebContext(http.MethodPost, `{}`, 1)
	rows := []*models.Account{
		{AccountId: 1, Uid: 1, Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT, Category: models.ACCOUNT_CATEGORY_CASH, Name: "钱包", Currency: "CNY", Balance: 3742},
		{AccountId: 2, Uid: 1, Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT, Category: models.ACCOUNT_CATEGORY_CASH, Name: "隐藏现金", Currency: "HKD", Balance: 3000, Hidden: true},
		{AccountId: 3, Uid: 2, Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT, Category: models.ACCOUNT_CATEGORY_CASH, Name: "其他用户", Currency: "CNY", Balance: 99999},
	}
	db := services.Accounts.UserDataDB(1).NewSession(c)
	defer db.Close()
	_, err := db.Insert(rows)
	require.NoError(t, err)
	writes := 0
	runtime := &conversationToolRuntime{c: c, state: &conversationState{}, zone: time.UTC, beforeWrite: func() error { writes++; return nil }}
	result, err := runtime.execute("list_accounts", json.RawMessage(`{}`))
	require.NoError(t, err)
	require.Contains(t, result.Details, "钱包")
	require.NotContains(t, result.Details, "隐藏现金")
	require.NotContains(t, result.Details, "其他用户")
	_, err = runtime.execute("set_account_balance", json.RawMessage(`{"ordinal":1,"account_id":"2","currency":"CNY","amount":"6000"}`))
	require.Error(t, err)
	_, err = runtime.execute("set_account_balance", json.RawMessage(`{"ordinal":1,"currency":"HKD","amount":"6000"}`))
	require.Error(t, err)
	require.Zero(t, writes)
	result, err = runtime.execute("set_account_balance", json.RawMessage(`{"ordinal":1,"currency":"CNY","amount":"6000"}`))
	require.NoError(t, err)
	require.NotEmpty(t, result.Verified, "%+v", result)
	account, err := services.Accounts.GetAccountByAccountId(c, 1, "1")
	require.NoError(t, err)
	require.Equal(t, int64(600000), account.Balance)
	_, err = runtime.execute("set_account_balance", json.RawMessage(`{"ordinal":1,"currency":"CNY","amount":"7000"}`))
	require.Error(t, err)
	require.Equal(t, 1, writes)
	result, err = runtime.execute("list_accounts", json.RawMessage(`{"include_hidden":true}`))
	require.NoError(t, err)
	require.Contains(t, result.Details, "隐藏现金")
	require.Contains(t, result.Reply, "HKD")
	runtime.attempted = false
	require.NoError(t, services.Accounts.ModifyAccountBalance(c, 1, "1", 601000))
	_, err = runtime.execute("set_account_balance", json.RawMessage(`{"ordinal":1,"currency":"CNY","amount":"7000"}`))
	require.Error(t, err)
	require.Equal(t, 1, writes)
	count, err := db.Count(new(models.Transaction))
	require.NoError(t, err)
	require.Zero(t, count)
}

func TestConversationMoneyRejectsImpreciseOrNonDecimalAmounts(t *testing.T) {
	for _, value := range []string{"1.001", "1e3", "1/2", "1000000000", "NaN"} {
		_, err := conversationAmount(value)
		require.Error(t, err, value)
	}
	for value, expected := range map[string]int64{"0": 0, "-1.01": -101, "6000": 600000, "37.42": 3742} {
		actual, err := conversationAmount(value)
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	}
}

func TestConversationQueryPaginatesRealTransactionsWithinLocalDay(t *testing.T) {
	config := &settings.Config{DatabaseConfig: &settings.DatabaseConfig{DatabaseType: settings.Sqlite3DbType, DatabasePath: filepath.Join(t.TempDir(), "query.db")}}
	config.UuidGeneratorType = settings.InternalUuidGeneratorType
	previousConfig := settings.Container.GetCurrentConfig()
	settings.SetCurrentConfig(config)
	t.Cleanup(func() { settings.SetCurrentConfig(previousConfig) })
	require.NoError(t, uuid.InitializeUuidGenerator(config))
	require.NoError(t, datastore.InitializeDataStore(config))
	require.NoError(t, datastore.Container.UserStore.SyncStructs(new(models.User)))
	require.NoError(t, datastore.Container.UserDataStore.SyncStructs(new(models.Account), new(models.Transaction), new(models.TransactionCategory), new(models.TransactionTag), new(models.TransactionTagIndex), new(models.TransactionPictureInfo)))
	v := binding.Validator.Engine().(*validator.Validate)
	require.NoError(t, v.RegisterValidation("validTagFilter", validators.ValidTagFilter))
	require.NoError(t, v.RegisterValidation("validAmountFilter", validators.ValidAmountFilter))
	c := newAssetPlanTestWebContext(http.MethodPost, `{}`, 1)
	db := services.Accounts.UserDataDB(1).NewSession(c)
	defer db.Close()
	userdb := services.Users.UserDB().NewSession(c)
	defer userdb.Close()
	_, err := userdb.Insert(&models.User{Uid: 1, Username: "test", DefaultCurrency: "CNY", TransactionEditScope: models.TRANSACTION_EDIT_SCOPE_ALL})
	require.NoError(t, err)
	_, err = db.Insert(&models.Account{AccountId: 1, Uid: 1, Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT, Category: models.ACCOUNT_CATEGORY_CASH, Name: "钱包", Currency: "CNY"})
	require.NoError(t, err)
	_, err = db.Insert(&models.TransactionCategory{CategoryId: 1, Uid: 1, ParentCategoryId: 2, Name: "餐饮", Type: models.CATEGORY_TYPE_EXPENSE})
	require.NoError(t, err)
	day := time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC).Unix() * 1000
	for i := int64(0); i < 51; i++ {
		_, err = db.Insert(&models.Transaction{TransactionId: i + 1, Uid: 1, Type: models.TRANSACTION_DB_TYPE_EXPENSE, CategoryId: 1, AccountId: 1, Amount: 100, TransactionTime: day + i})
		require.NoError(t, err)
	}
	_, err = db.Insert(&models.Transaction{TransactionId: 52, Uid: 1, Type: models.TRANSACTION_DB_TYPE_EXPENSE, CategoryId: 1, AccountId: 1, Amount: 9999, TransactionTime: day - 1})
	require.NoError(t, err)
	runtime := &conversationToolRuntime{c: c, state: &conversationState{}, zone: time.UTC}
	result, err := runtime.execute("query_transactions", json.RawMessage(`{"type":3,"start_date":"2026-09-13","end_date":"2026-09-13"}`))
	require.NoError(t, err)
	require.Contains(t, result.Reply, "51.00 CNY")
	data := result.Data.(map[string]any)
	require.Equal(t, 51, data["total_count"])
	require.Len(t, data["transactions"], 50)
	result, err = runtime.execute("query_transactions", json.RawMessage(`{"type":3,"start_date":"2026-09-13","end_date":"2026-09-13","offset":50}`))
	require.NoError(t, err)
	require.Len(t, result.Data.(map[string]any)["transactions"], 1)
	_, err = db.Insert(&models.TransactionCategory{CategoryId: 2, Uid: 1, Name: "支出", Type: models.CATEGORY_TYPE_EXPENSE})
	require.NoError(t, err)
	runtime.beforeWrite = func() error { return nil }
	runtime.messageID = "create-test"
	result, err = runtime.execute("create_transaction", json.RawMessage(`{"type":3,"account_id":"1","category_id":"1","currency":"CNY","amount":"30.50","comment":"午饭"}`))
	require.NoError(t, err)
	require.False(t, result.Unknown)
	require.NotEmpty(t, result.Verified, "%+v", result)
	account, err := services.Accounts.GetAccountByAccountId(c, 1, "1")
	require.NoError(t, err)
	require.Zero(t, account.Balance, "cash flow must not change the account balance")

}
