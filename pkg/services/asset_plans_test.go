package services

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mayswind/financially_free/pkg/core"
	"github.com/mayswind/financially_free/pkg/datastore"
	"github.com/mayswind/financially_free/pkg/errs"
	"github.com/mayswind/financially_free/pkg/models"
	"github.com/mayswind/financially_free/pkg/settings"
)

func TestAssetPlanServiceCRUDAndUserIsolation(t *testing.T) {
	config := &settings.Config{
		DatabaseConfig: &settings.DatabaseConfig{
			DatabaseType: settings.Sqlite3DbType,
			DatabasePath: filepath.Join(t.TempDir(), "asset-plan-test.db"),
		},
	}
	require.NoError(t, datastore.InitializeDataStore(config))
	require.NoError(t, datastore.Container.UserDataStore.SyncStructs(new(models.AssetPlan)))
	require.NoError(t, datastore.Container.UserDataStore.SyncStructs(new(models.AssetPlan)))

	service := AssetPlans
	context := core.NewNullContext()
	plan := &models.AssetPlan{
		Uid:          101,
		TargetAmount: 2000,
		TargetDate:   20261231,
		StartAmount:  1000,
		StartDate:    20260101,
		Currency:     "CNY",
	}
	plan.SetExcludedAccountIds([]int64{12, 34})

	require.NoError(t, service.CreateAssetPlan(context, plan))

	actual, err := service.GetAssetPlanByUid(context, 101)
	require.NoError(t, err)
	require.NotNil(t, actual)
	assert.Equal(t, int64(1000), actual.StartAmount)
	assert.Equal(t, []int64{12, 34}, actual.GetExcludedAccountIds())

	err = service.CreateAssetPlan(context, plan)
	assert.ErrorIs(t, err, errs.ErrAssetPlanAlreadyExists)

	actual.TargetAmount = 2500
	actual.TargetDate = 20270131
	actual.StartAmount = 9999
	actual.StartDate = 20260202
	actual.Currency = "USD"
	actual.SetExcludedAccountIds([]int64{56})
	require.NoError(t, service.ModifyAssetPlan(context, actual))

	modified, err := service.GetAssetPlanByUid(context, 101)
	require.NoError(t, err)
	require.NotNil(t, modified)
	assert.Equal(t, int64(2500), modified.TargetAmount)
	assert.Equal(t, int32(20270131), modified.TargetDate)
	assert.Equal(t, int64(1000), modified.StartAmount)
	assert.Equal(t, int32(20260101), modified.StartDate)
	assert.Equal(t, "CNY", modified.Currency)
	assert.Equal(t, []int64{12, 34}, modified.GetExcludedAccountIds())

	otherUserPlan := *modified
	otherUserPlan.Uid = 202
	err = service.ModifyAssetPlan(context, &otherUserPlan)
	assert.ErrorIs(t, err, errs.ErrAssetPlanNotFound)

	err = service.DeleteAssetPlan(context, 202)
	assert.ErrorIs(t, err, errs.ErrAssetPlanNotFound)

	require.NoError(t, service.DeleteAssetPlan(context, 101))
	actual, err = service.GetAssetPlanByUid(context, 101)
	require.NoError(t, err)
	assert.Nil(t, actual)

	require.NoError(t, service.CreateAssetPlan(context, plan))
}
