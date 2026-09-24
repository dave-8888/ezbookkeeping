package api

import (
	"math"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mayswind/financially_free/pkg/core"
	"github.com/mayswind/financially_free/pkg/errs"
	"github.com/mayswind/financially_free/pkg/models"
)

type fakeAssetPlanStore struct {
	plan         *models.AssetPlan
	requestedUid int64
}

func (s *fakeAssetPlanStore) GetAssetPlanByUid(_ core.Context, uid int64) (*models.AssetPlan, error) {
	s.requestedUid = uid
	return s.plan, nil
}

func (s *fakeAssetPlanStore) CreateAssetPlan(_ core.Context, plan *models.AssetPlan) error {
	s.plan = plan
	return nil
}

func (s *fakeAssetPlanStore) ModifyAssetPlan(_ core.Context, plan *models.AssetPlan) error {
	s.plan = plan
	return nil
}

func (s *fakeAssetPlanStore) DeleteAssetPlan(_ core.Context, uid int64) error {
	s.requestedUid = uid
	s.plan = nil
	return nil
}

type fakeAssetPlanNetWorthProvider struct {
	current                  int64
	values                   map[int32]int64
	resolvedExcludedAccounts []int64
}

func (p *fakeAssetPlanNetWorthProvider) CalculateCurrentNetWorth(_ core.Context, _ int64, _ string, _ []int64) (int64, error) {
	return p.current, nil
}

func (p *fakeAssetPlanNetWorthProvider) ResolveExcludedAccountIds(_ core.Context, _ int64, _ []string) ([]int64, error) {
	return p.resolvedExcludedAccounts, nil
}

func (p *fakeAssetPlanNetWorthProvider) CalculateNetWorth(_ core.Context, _ int64, date int32, _ *time.Location, _ string, _ []int64) (int64, error) {
	return p.values[date], nil
}

type fakeAssetPlanUserProvider struct {
	user *models.User
}

func (p *fakeAssetPlanUserProvider) GetUserById(_ core.Context, _ int64) (*models.User, error) {
	return p.user, nil
}

func TestAssetPlanGetUsesAuthenticatedUserOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ginContext, _ := gin.CreateTestContext(httptest.NewRecorder())
	webContext := core.WrapWebContext(ginContext)
	webContext.SetTokenClaims(&core.UserTokenClaims{
		Uid:  42,
		Type: core.USER_TOKEN_TYPE_NORMAL,
	})

	store := &fakeAssetPlanStore{}
	api := &AssetPlansApi{
		assetPlans: store,
		now:        time.Now,
	}

	result, err := api.AssetPlanGetHandler(webContext)
	require.Nil(t, err)
	assert.Nil(t, result)
	assert.Equal(t, int64(42), store.requestedUid)
}

func TestAssetPlanCreateValidatesDateAndAmountAndFreezesScope(t *testing.T) {
	tests := []struct {
		name          string
		body          string
		expectedError error
	}{
		{
			name:          "target date is today",
			body:          `{"targetAmount":200,"targetDate":"2026-01-01"}`,
			expectedError: errs.ErrAssetPlanTargetDateInvalid,
		},
		{
			name:          "target amount equals start",
			body:          `{"targetAmount":100,"targetDate":"2026-01-03"}`,
			expectedError: errs.ErrAssetPlanTargetAmountInvalid,
		},
		{
			name: "valid",
			body: `{"targetAmount":200,"targetDate":"2026-01-03","excludedAccountIds":["9"]}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := &fakeAssetPlanStore{}
			netWorth := &fakeAssetPlanNetWorthProvider{
				values:                   map[int32]int64{20260101: 100},
				resolvedExcludedAccounts: []int64{91, 92},
			}
			api := &AssetPlansApi{
				assetPlans: store,
				users: &fakeAssetPlanUserProvider{
					user: &models.User{DefaultCurrency: "CNY"},
				},
				netWorth: netWorth,
				now: func() time.Time {
					return time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
				},
			}
			webContext := newAssetPlanTestWebContext(http.MethodPost, test.body, 42)

			result, finalErr := api.AssetPlanCreateHandler(webContext)

			if test.expectedError != nil {
				require.NotNil(t, finalErr)
				assert.Equal(t, test.expectedError.(*errs.Error).Code(), finalErr.Code())
				assert.Nil(t, result)
				return
			}

			require.Nil(t, finalErr)
			response, ok := result.(*models.AssetPlanInfoResponse)
			require.True(t, ok)
			require.NotNil(t, store.plan)
			assert.Equal(t, int64(100), store.plan.StartAmount)
			assert.Equal(t, int32(20260101), store.plan.StartDate)
			assert.Equal(t, "CNY", store.plan.Currency)
			assert.Equal(t, []int64{91, 92}, store.plan.GetExcludedAccountIds())
			assert.Equal(t, int64(100), response.CurrentActualNetWorth)
		})
	}
}

func TestBuildAssetPlanResponseUsesTargetDateActualForExpiredStatus(t *testing.T) {
	plan := &models.AssetPlan{
		Uid:          42,
		StartAmount:  100,
		StartDate:    20260101,
		TargetAmount: 200,
		TargetDate:   20260111,
		Currency:     "CNY",
	}
	netWorth := &fakeAssetPlanNetWorthProvider{
		values: map[int32]int64{
			20260111: 190,
			20260112: 250,
		},
	}
	api := &AssetPlansApi{
		netWorth: netWorth,
		now: func() time.Time {
			return time.Date(2026, 1, 12, 12, 0, 0, 0, time.UTC)
		},
	}

	response, err := api.buildAssetPlanResponse(core.NewNullContext(), plan, time.UTC, nil)
	require.NoError(t, err)
	assert.Equal(t, models.ASSET_PLAN_STATUS_NOT_ACHIEVED, response.Status)
	assert.Equal(t, int64(250), response.CurrentActualNetWorth)
	require.NotNil(t, response.TargetDateActualNetWorth)
	assert.Equal(t, int64(190), *response.TargetDateActualNetWorth)
	assert.Equal(t, "100.00", response.CompletionPercentage)
}

func TestBuildAssetPlanResponseTargetStatuses(t *testing.T) {
	tests := []struct {
		name     string
		actual   int64
		expected string
	}{
		{name: "achieved", actual: 200, expected: models.ASSET_PLAN_STATUS_ACHIEVED},
		{name: "not achieved", actual: 199, expected: models.ASSET_PLAN_STATUS_NOT_ACHIEVED},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plan := &models.AssetPlan{
				Uid:          42,
				StartAmount:  100,
				StartDate:    20260101,
				TargetAmount: 200,
				TargetDate:   20260103,
				Currency:     "CNY",
			}
			api := &AssetPlansApi{
				netWorth: &fakeAssetPlanNetWorthProvider{
					values: map[int32]int64{20260103: test.actual},
				},
				now: func() time.Time {
					return time.Date(2026, 1, 3, 12, 0, 0, 0, time.UTC)
				},
			}

			response, err := api.buildAssetPlanResponse(core.NewNullContext(), plan, time.UTC, nil)
			require.NoError(t, err)
			assert.Equal(t, test.expected, response.Status)
		})
	}
}

func TestBuildAssetPlanResponseComparisonStatuses(t *testing.T) {
	tests := []struct {
		name       string
		actual     int64
		expected   string
		difference int64
	}{
		{name: "ahead", actual: 160, expected: models.ASSET_PLAN_COMPARISON_STATUS_AHEAD, difference: 10},
		{name: "behind", actual: 140, expected: models.ASSET_PLAN_COMPARISON_STATUS_BEHIND, difference: -10},
		{name: "on track", actual: 150, expected: models.ASSET_PLAN_COMPARISON_STATUS_ON_TRACK, difference: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plan := &models.AssetPlan{
				Uid:          42,
				StartAmount:  100,
				StartDate:    20260101,
				TargetAmount: 200,
				TargetDate:   20260103,
				Currency:     "CNY",
			}
			api := &AssetPlansApi{
				netWorth: &fakeAssetPlanNetWorthProvider{
					values: map[int32]int64{20260102: test.actual},
				},
				now: func() time.Time {
					return time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC)
				},
			}

			response, err := api.buildAssetPlanResponse(core.NewNullContext(), plan, time.UTC, nil)
			require.NoError(t, err)
			assert.Equal(t, test.expected, response.ComparisonStatus)
			assert.Equal(t, test.difference, response.DifferenceAmount)
			assert.Equal(t, models.ASSET_PLAN_STATUS_IN_PROGRESS, response.Status)
		})
	}
}

func TestBuildAssetPlanResponseRejectsDifferenceOverflow(t *testing.T) {
	plan := &models.AssetPlan{
		Uid:          42,
		StartAmount:  math.MinInt64,
		StartDate:    20260101,
		TargetAmount: math.MaxInt64,
		TargetDate:   20260103,
		Currency:     "CNY",
	}
	api := &AssetPlansApi{
		netWorth: &fakeAssetPlanNetWorthProvider{
			values: map[int32]int64{20260101: math.MaxInt64},
		},
		now: func() time.Time {
			return time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
		},
	}

	response, err := api.buildAssetPlanResponse(core.NewNullContext(), plan, time.UTC, nil)
	require.ErrorIs(t, err, errs.ErrNumberInvalid)
	assert.Nil(t, response)
}

func TestConvertAssetPlanAmountUsesExactDecimalRatesAndTruncates(t *testing.T) {
	fromRate, ok := new(big.Rat).SetString("7.3")
	require.True(t, ok)
	toRate, ok := new(big.Rat).SetString("1")
	require.True(t, ok)

	converted, ok := convertAssetPlanAmount(1000, fromRate, toRate)
	require.True(t, ok)
	assert.Equal(t, int64(136), converted)

	converted, ok = convertAssetPlanAmount(-1000, fromRate, toRate)
	require.True(t, ok)
	assert.Equal(t, int64(-136), converted)
}

func newAssetPlanTestWebContext(method string, body string, uid int64) *core.WebContext {
	gin.SetMode(gin.TestMode)
	ginContext, _ := gin.CreateTestContext(httptest.NewRecorder())
	ginContext.Request = httptest.NewRequest(method, "/api/v1/asset_plans/test.json", strings.NewReader(body))
	ginContext.Request.Header.Set("Content-Type", "application/json")
	ginContext.Request.Header.Set(core.ClientTimezoneOffsetHeaderName, "0")
	webContext := core.WrapWebContext(ginContext)
	webContext.SetTokenClaims(&core.UserTokenClaims{
		Uid:  uid,
		Type: core.USER_TOKEN_TYPE_NORMAL,
	})
	return webContext
}

func (s *fakeAssetPlanStore) SaveActualTracking(_ core.Context, _ int64, tracking string) error {
	s.plan.ActualTracking = tracking
	return nil
}
