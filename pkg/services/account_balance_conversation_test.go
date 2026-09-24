package services

import (
	"path/filepath"
	"testing"

	"github.com/mayswind/financially_free/pkg/core"
	"github.com/mayswind/financially_free/pkg/datastore"
	"github.com/mayswind/financially_free/pkg/errs"
	"github.com/mayswind/financially_free/pkg/models"
	"github.com/mayswind/financially_free/pkg/settings"
	"github.com/stretchr/testify/require"
)

func TestConversationBalanceCompareAndSet(t *testing.T) {
	config := &settings.Config{DatabaseConfig: &settings.DatabaseConfig{DatabaseType: settings.Sqlite3DbType, DatabasePath: filepath.Join(t.TempDir(), "balance.db")}}
	require.NoError(t, datastore.InitializeDataStore(config))
	require.NoError(t, datastore.Container.UserDataStore.SyncStructs(new(models.Account), new(models.Transaction)))
	ctx := core.NewNullContext()
	account := &models.Account{AccountId: 1, Uid: 1, Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT, Category: models.ACCOUNT_CATEGORY_CASH, Name: "test", Currency: "CNY", Balance: 500000}
	_, err := Accounts.UserDataDB(1).NewSession(ctx).Insert(account)
	require.NoError(t, err)
	require.NoError(t, Accounts.ModifyAccountBalance(ctx, 1, "1", 600000, 500000))
	require.ErrorIs(t, Accounts.ModifyAccountBalance(ctx, 1, "1", 700000, 500000), errs.ErrAccountBalanceChanged)
	require.ErrorIs(t, Accounts.ModifyAccountBalance(ctx, 2, "1", 900000, 600000), errs.ErrAccountNotFound)
	current, err := Accounts.GetAccountByAccountId(ctx, 1, "1")
	require.NoError(t, err)
	require.Equal(t, int64(600000), current.Balance)
	require.NoError(t, Accounts.ModifyAccountBalance(ctx, 1, "1", 0, 600000))
	require.NoError(t, Accounts.ModifyAccountBalance(ctx, 1, "1", -100, 0))
	// Existing callers can still perform an absolute adjustment without a precondition.
	require.NoError(t, Accounts.ModifyAccountBalance(ctx, 1, "1", 200))
	count, err := Accounts.UserDataDB(1).NewSession(ctx).Count(new(models.Transaction))
	require.NoError(t, err)
	require.Zero(t, count, "balance calibration must not create income or expense")
}
