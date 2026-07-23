package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAndFormatAssetPlanDate(t *testing.T) {
	location, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)

	value, err := ParseAssetPlanDate("2028-02-29", location)
	require.NoError(t, err)
	assert.Equal(t, int32(20280229), value)
	assert.Equal(t, "2028-02-29", FormatAssetPlanDate(value))

	_, err = ParseAssetPlanDate("2027-02-29", location)
	assert.Error(t, err)

	_, err = ParseAssetPlanDate("2028-2-29", location)
	assert.Error(t, err)
}

func TestCalculateAssetPlanValue(t *testing.T) {
	location, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)

	plan := &AssetPlan{
		StartAmount:  100,
		StartDate:    20260307,
		TargetAmount: 200,
		TargetDate:   20260309,
	}

	startValue, err := CalculateAssetPlanValue(plan, 20260307, location)
	require.NoError(t, err)
	assert.Equal(t, int64(100), startValue)

	middleValue, err := CalculateAssetPlanValue(plan, 20260308, location)
	require.NoError(t, err)
	assert.Equal(t, int64(150), middleValue)

	targetValue, err := CalculateAssetPlanValue(plan, 20260309, location)
	require.NoError(t, err)
	assert.Equal(t, int64(200), targetValue)

	afterTargetValue, err := CalculateAssetPlanValue(plan, 20260310, location)
	require.NoError(t, err)
	assert.Equal(t, int64(200), afterTargetValue)
}

func TestCalculateAssetPlanValueAcrossLeapDayWithStableRounding(t *testing.T) {
	plan := &AssetPlan{
		StartAmount:  -100,
		StartDate:    20280228,
		TargetAmount: 101,
		TargetDate:   20280302,
	}

	leapDayValue, err := CalculateAssetPlanValue(plan, 20280229, time.UTC)
	require.NoError(t, err)
	assert.Equal(t, int64(-33), leapDayValue)

	dayBeforeTargetValue, err := CalculateAssetPlanValue(plan, 20280301, time.UTC)
	require.NoError(t, err)
	assert.Equal(t, int64(34), dayBeforeTargetValue)

	targetValue, err := CalculateAssetPlanValue(plan, 20280302, time.UTC)
	require.NoError(t, err)
	assert.Equal(t, int64(101), targetValue)
}

func TestCalculateAssetPlanValueDoesNotOverflowIntermediateResult(t *testing.T) {
	plan := &AssetPlan{
		StartAmount:  -9_000_000_000_000_000_000,
		StartDate:    20260101,
		TargetAmount: 9_000_000_000_000_000_000,
		TargetDate:   20260103,
	}

	value, err := CalculateAssetPlanValue(plan, 20260102, time.UTC)
	require.NoError(t, err)
	assert.Equal(t, int64(0), value)
}

func TestCalculateAssetPlanCompletionPercentage(t *testing.T) {
	assert.Equal(t, "0.00", CalculateAssetPlanCompletionPercentage(100, 200, 50))
	assert.Equal(t, "0.00", CalculateAssetPlanCompletionPercentage(100, 200, 100))
	assert.Equal(t, "50.00", CalculateAssetPlanCompletionPercentage(100, 200, 150))
	assert.Equal(t, "100.00", CalculateAssetPlanCompletionPercentage(100, 200, 200))
	assert.Equal(t, "100.00", CalculateAssetPlanCompletionPercentage(100, 200, 250))
}

func TestAssetPlanExcludedAccountIds(t *testing.T) {
	plan := &AssetPlan{}
	plan.SetExcludedAccountIds([]int64{12, 34})

	assert.Equal(t, "12,34", plan.ExcludedAccountIds)
	assert.Equal(t, []int64{12, 34}, plan.GetExcludedAccountIds())
}
