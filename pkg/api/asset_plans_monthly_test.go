package api

import (
	"github.com/mayswind/financially_free/pkg/core"
	"github.com/mayswind/financially_free/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestMonthlyPlanLedgerComparison(t *testing.T) {
	api := &AssetPlansApi{netWorth: &fakeAssetPlanNetWorthProvider{values: map[int32]int64{20231231: 1000000, 20240131: 1400000, 20240229: 1300000, 20240312: 1600000}}, now: func() time.Time { return time.Date(2024, 3, 12, 12, 0, 0, 0, time.UTC) }}
	plan := &models.AssetPlan{Uid: 42, Currency: "CNY", MonthlySchedule: `{"startMonth":"2024-01","endMonth":"2024-04","stages":[{"amount":500000,"repeat":4}]}`}
	response, err := api.buildMonthlyAssetPlanResponse(core.NewNullContext(), plan, time.UTC)
	require.NoError(t, err)
	monthly := response.Monthly
	assert.Equal(t, "2023-12-31", monthly.BaselineDate)
	assert.Equal(t, int64(1000000), *monthly.BaselineAmount)
	assert.Equal(t, int64(-100000), *monthly.Months[1].Actual)
	assert.Equal(t, int64(300000), *monthly.Months[1].CumulativeActual)
	assert.Equal(t, "2024-02-29", monthly.Months[1].ActualThrough)
	assert.Equal(t, int64(600000), *monthly.Months[2].CumulativeActual)
	assert.Equal(t, int64(-900000), *monthly.Months[2].Difference)
	assert.Equal(t, "2024-03-12", monthly.Months[2].ActualThrough)
	assert.Nil(t, monthly.Months[3].Actual)
	assert.Nil(t, monthly.Months[3].Difference)
	// Once ended, never include balances after the end month.
	api.now = func() time.Time { return time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC) }
	response, err = api.buildMonthlyAssetPlanResponse(core.NewNullContext(), plan, time.UTC)
	require.NoError(t, err)
	assert.Equal(t, "2024-04-30", response.Monthly.Months[3].ActualThrough)
}
func TestMonthlyFuturePlanDoesNotReadLedger(t *testing.T) {
	api := &AssetPlansApi{now: func() time.Time { return time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC) }}
	plan := &models.AssetPlan{MonthlySchedule: `{"startMonth":"2026-02","endMonth":"2026-02","stages":[{"amount":100,"repeat":1}]}`}
	response, err := api.buildMonthlyAssetPlanResponse(core.NewNullContext(), plan, time.UTC)
	require.NoError(t, err)
	assert.Nil(t, response.Monthly.BaselineAmount)
	assert.Nil(t, response.Monthly.Months[0].Actual)
}
func TestMonthlyCreateAndReplaceLegacy(t *testing.T) {
	store := &fakeAssetPlanStore{}
	api := &AssetPlansApi{assetPlans: store, users: &fakeAssetPlanUserProvider{user: &models.User{DefaultCurrency: "CNY"}}, netWorth: &fakeAssetPlanNetWorthProvider{values: map[int32]int64{}, resolvedExcludedAccounts: []int64{9}}, now: func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) }}
	body := `{"startMonth":"2026-02","endMonth":"2026-03","stages":[{"amount":500000,"repeat":2}],"excludedAccountIds":["9"]}`
	result, err := api.AssetPlanCreateHandler(newAssetPlanTestWebContext(http.MethodPost, body, 42))
	require.Nil(t, err)
	response := result.(*models.AssetPlanInfoResponse)
	assert.Equal(t, int64(1000000), response.Monthly.TotalPlanned)
	assert.Equal(t, int32(20260201), store.plan.StartDate)
	assert.Equal(t, []int64{9}, store.plan.GetExcludedAccountIds())
	store.plan.MonthlySchedule = ""
	store.plan.StartAmount = 3155932
	result, err = api.AssetPlanModifyHandler(newAssetPlanTestWebContext(http.MethodPost, body, 42))
	require.Nil(t, err)
	assert.NotNil(t, result.(*models.AssetPlanInfoResponse).Monthly)
	assert.Equal(t, "CNY", store.plan.Currency)
	assert.Equal(t, []int64{9}, store.plan.GetExcludedAccountIds())
	// Invalid schedule must not mutate the saved plan.
	before := store.plan.MonthlySchedule
	_, err = api.AssetPlanModifyHandler(newAssetPlanTestWebContext(http.MethodPost, `{"startMonth":"2026-02","endMonth":"2026-03","stages":[{"amount":100,"repeat":1}]}`, 42))
	require.NotNil(t, err)
	assert.Equal(t, before, store.plan.MonthlySchedule)
}

func TestGoalPlanActualsAreIndependentOfForecast(t *testing.T) {
	api := &AssetPlansApi{netWorth: &fakeAssetPlanNetWorthProvider{current: 950000, values: map[int32]int64{20260930: 1000000, 20261001: 1100000, 20261002: 900000, 20261003: 950000}}, now: func() time.Time { return time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC) }}
	plan := &models.AssetPlan{Uid: 42, Currency: "CNY", ActualTracking: `{"date":"2026-09-30","amount":1000000,"snapshots":{"2026-10-01":1100000,"2026-10-02":900000}}`, MonthlySchedule: `{"version":2,"startMonth":"2026-10","endMonth":"2026-10","finalGoal":{"amount":2000000,"date":"2026-10-31"},"stages":[{"name":"Save","amount":100000,"repeat":1,"startDate":"2026-10-01","depositDay":1}],"expenses":[{"name":"Phone","amount":500000,"date":"2026-10-02"}]}`}
	response, err := api.buildMonthlyAssetPlanResponse(core.NewNullContext(), plan, time.UTC)
	require.NoError(t, err)
	assert.Equal(t, int64(950000), *response.Monthly.CurrentNetWorth)
	assert.Equal(t, int64(-50000), *response.Monthly.Months[0].CumulativeActual)
	require.NoError(t, api.addMonthlyPlanDays(core.NewNullContext(), plan, response.Monthly, "2026-10", time.UTC))
	require.Len(t, response.Monthly.Days, 31)
	assert.Equal(t, int64(-100000), *response.Monthly.Days[1].CumulativeActual)
	assert.Equal(t, int64(-400000), response.Monthly.Days[1].CumulativePlanned)
	assert.Nil(t, response.Monthly.Days[3].Actual)
	assert.Equal(t, *response.Monthly.Months[0].CumulativeActual, *response.Monthly.Days[2].CumulativeActual)
	plan.MonthlySchedule = strings.Replace(plan.MonthlySchedule, `"amount":500000`, `"amount":800000`, 1)
	changed, err := api.buildMonthlyAssetPlanResponse(core.NewNullContext(), plan, time.UTC)
	require.NoError(t, err)
	assert.Equal(t, response.Monthly.Months[0].CumulativeActual, changed.Monthly.Months[0].CumulativeActual)
	require.Error(t, api.addMonthlyPlanDays(core.NewNullContext(), plan, changed.Monthly, "2027-01", time.UTC))
}

func TestGoalPlanSaveAndFutureProjection(t *testing.T) {
	store := &fakeAssetPlanStore{}
	api := &AssetPlansApi{assetPlans: store, users: &fakeAssetPlanUserProvider{user: &models.User{DefaultCurrency: "CNY"}}, netWorth: &fakeAssetPlanNetWorthProvider{current: 2000000, values: map[int32]int64{20260912: 1700000}}, now: func() time.Time { return time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC) }}
	body := `{"version":2,"startMonth":"2026-10","endMonth":"2026-12","finalGoal":{"amount":5000000,"date":"2026-12-31"},"goals":[{"id":"phone","name":"Phone","amount":1200000,"date":"2026-12-20"}],"stages":[{"name":"Save","amount":1400000,"repeat":3,"startDate":"2026-10-01","depositDay":1,"goalId":"phone"}],"expenses":[{"name":"Phone","amount":1200000,"date":"2026-12-20","goalId":"phone"}]}`
	result, err := api.AssetPlanCreateHandler(newAssetPlanTestWebContext(http.MethodPost, body, 42))
	require.Nil(t, err)
	monthly := result.(*models.AssetPlanInfoResponse).Monthly
	assert.Equal(t, int64(2000000), *monthly.ProjectionBase)
	assert.Equal(t, int64(2000000), *monthly.BaselineAmount)
	assert.Nil(t, monthly.Months[0].Actual)
	assert.Equal(t, int64(3000000), monthly.TotalPlanned)
	reloaded, loadErr := api.buildMonthlyAssetPlanResponse(core.NewNullContext(), store.plan, time.UTC)
	require.NoError(t, loadErr)
	assert.Equal(t, monthly.Goals, reloaded.Monthly.Goals)
	before := store.plan.MonthlySchedule
	_, err = api.AssetPlanModifyHandler(newAssetPlanTestWebContext(http.MethodPost, strings.Replace(body, `"goalId":"phone"`, `"goalId":"missing"`, 1), 42))
	require.NotNil(t, err)
	assert.Equal(t, before, store.plan.MonthlySchedule)
}

func TestGoalPlanBalanceBaselineAndSnapshots(t *testing.T) {
	provider := &fakeAssetPlanNetWorthProvider{current: 8312785, values: map[int32]int64{20260831: 3153490, 20260913: 4909707}}
	now := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)
	api := &AssetPlansApi{netWorth: provider, now: func() time.Time { return now }}
	plan := &models.AssetPlan{Uid: 42, Currency: "CNY", MonthlySchedule: `{"version":2,"startMonth":"2026-09","endMonth":"2026-10","finalGoal":{"amount":10000000,"date":"2026-10-31"},"stages":[{"name":"Save","amount":1400000,"repeat":2,"startDate":"2026-09-01","depositDay":1}]}`}
	response, err := api.buildMonthlyAssetPlanResponse(core.NewNullContext(), plan, time.UTC)
	require.NoError(t, err)
	require.NotEmpty(t, plan.ActualTracking)
	assert.Equal(t, "2026-09-13", response.Monthly.BaselineDate)
	assert.Equal(t, int64(8312785), *response.Monthly.BaselineAmount)
	assert.Equal(t, int64(0), *response.Monthly.Months[0].CumulativeActual)
	assert.Equal(t, int64(0), *response.Monthly.Months[0].Difference)
	assert.Equal(t, int64(1400000), response.Monthly.TotalPlanned)
	// Refresh observes quick balance edits, with no ledger changes and no baseline reset.
	provider.current += 10000
	now = now.AddDate(0, 0, 2)
	response, err = api.buildMonthlyAssetPlanResponse(core.NewNullContext(), plan, time.UTC)
	require.NoError(t, err)
	assert.Equal(t, int64(8312785), *response.Monthly.BaselineAmount)
	assert.Equal(t, int64(10000), *response.Monthly.Months[0].CumulativeActual)
	require.NoError(t, api.addMonthlyPlanDays(core.NewNullContext(), plan, response.Monthly, "2026-09", time.UTC))
	assert.Nil(t, response.Monthly.Days[11].CumulativeActual)
	assert.Equal(t, int64(0), *response.Monthly.Days[12].CumulativeActual)
	assert.Nil(t, response.Monthly.Days[13].CumulativeActual)
	assert.Equal(t, int64(10000), *response.Monthly.Days[14].CumulativeActual)
	assert.Nil(t, response.Monthly.Days[14].Actual) // No snapshot on the previous day.
	// A scheduled deposit becomes due on October 1, without inventing September's closing balance.
	now = time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	response, err = api.buildMonthlyAssetPlanResponse(core.NewNullContext(), plan, time.UTC)
	require.NoError(t, err)
	assert.Equal(t, int64(1400000), response.Monthly.PlannedToDate)
	assert.Equal(t, int64(-1390000), *response.Monthly.Months[1].Difference)
	assert.Nil(t, response.Monthly.Months[1].Actual)
	assert.Equal(t, int64(10000), *response.Monthly.Months[0].CumulativeActual)
	assert.Equal(t, "2026-09-15", response.Monthly.Months[0].ActualThrough)
}

func TestGoalPlanMonthlyLatestSnapshot(t *testing.T) {
	api := &AssetPlansApi{netWorth: &fakeAssetPlanNetWorthProvider{current: 1600000}, now: func() time.Time { return time.Date(2026, 12, 10, 12, 0, 0, 0, time.UTC) }}
	plan := &models.AssetPlan{Uid: 42, Currency: "CNY", ActualTracking: `{"date":"2026-09-13","amount":1000000,"snapshots":{"2026-09-12":9900000,"2026-09-13":1000000,"2026-09-28":1100000,"2026-09-20":1050000,"2026-10-31":1300000,"2026-10-29":1200000,"2026-12-31":9900000}}`, MonthlySchedule: `{"version":2,"startMonth":"2026-09","endMonth":"2027-01","finalGoal":{"amount":2000000,"date":"2027-01-31"},"stages":[{"name":"Save","amount":100000,"repeat":5,"startDate":"2026-09-01","depositDay":30}]}`}
	response, err := api.buildMonthlyAssetPlanResponse(core.NewNullContext(), plan, time.UTC)
	require.NoError(t, err)
	months := response.Monthly.Months
	assert.Equal(t, "2026-09-28", months[0].ActualThrough)
	assert.Equal(t, int64(100000), *months[0].CumulativeActual)
	assert.Equal(t, int64(100000), *months[0].Difference) // September 30 deposit is not yet due at the snapshot.
	assert.Equal(t, "2026-10-31", months[1].ActualThrough)
	assert.Equal(t, int64(300000), *months[1].CumulativeActual)
	assert.Nil(t, months[2].CumulativeActual) // No snapshot in November.
	assert.Equal(t, "2026-12-10", months[3].ActualThrough)
	assert.Equal(t, int64(600000), *months[3].CumulativeActual)
	assert.Nil(t, months[4].CumulativeActual)
}
