package models

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMonthlyScheduleExpansion(t *testing.T) {
	schedule := MonthlyAssetPlanSchedule{StartMonth: "2026-11", EndMonth: "2027-02", Stages: []AssetPlanStage{{Amount: 500000, Repeat: 2}, {Amount: 800000, Repeat: 2}}}
	months, err := schedule.Expand()
	require.NoError(t, err)
	assert.Equal(t, "2027-02", months[3].Month)
	assert.Equal(t, int64(2600000), months[3].CumulativePlanned)
	assert.Equal(t, int64(1800000), months[2].CumulativePlanned)
	for _, month := range months {
		assert.Nil(t, month.Actual)
	}
}
func TestMonthlyScheduleValidation(t *testing.T) {
	tests := []MonthlyAssetPlanSchedule{
		{StartMonth: "2026-1", EndMonth: "2026-02", Stages: []AssetPlanStage{{Amount: 1, Repeat: 2}}},
		{StartMonth: "2026-03", EndMonth: "2026-02", Stages: []AssetPlanStage{{Amount: 1, Repeat: 1}}},
		{StartMonth: "2026-01", EndMonth: "2036-01", Stages: []AssetPlanStage{{Amount: 1, Repeat: 121}}},
		{StartMonth: "2026-01", EndMonth: "2026-02", Stages: []AssetPlanStage{{Amount: 1, Repeat: 1}}},
		{StartMonth: "2026-01", EndMonth: "2026-02", Stages: []AssetPlanStage{{Amount: 1, Repeat: 3}}},
		{StartMonth: "2026-01", EndMonth: "2026-02", Stages: []AssetPlanStage{{Amount: -1, Repeat: 2}}},
		{StartMonth: "2026-01", EndMonth: "2026-02", Stages: []AssetPlanStage{{Amount: 0, Repeat: 2}}},
		{StartMonth: "2026-01", EndMonth: "2026-02", Stages: []AssetPlanStage{{Amount: 9007199254740991, Repeat: 2}}},
	}
	for _, schedule := range tests {
		_, err := schedule.Expand()
		require.Error(t, err)
	}
	_, err := (MonthlyAssetPlanSchedule{StartMonth: "2026-01", EndMonth: "2026-02", Stages: []AssetPlanStage{{Amount: 0, Repeat: 1}, {Amount: 1, Repeat: 1}}}).Expand()
	require.NoError(t, err)
}
