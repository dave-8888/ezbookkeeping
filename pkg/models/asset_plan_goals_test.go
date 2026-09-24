package models

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func phoneGoalPlan() MonthlyAssetPlanSchedule {
	return MonthlyAssetPlanSchedule{Version: 2, StartMonth: "2026-10", EndMonth: "2026-12", FinalGoal: &AssetPlanGoal{Amount: 3000000, Date: "2026-12-31"}, Goals: []AssetPlanGoal{{ID: "phone", Name: "Phone", Amount: 1200000, Date: "2026-12-20"}}, Stages: []AssetPlanStage{{Name: "Long term", Amount: 1000000, Repeat: 3, StartDate: "2026-10-01", DepositDay: 1}, {Name: "Phone", Amount: 400000, Repeat: 3, StartDate: "2026-10-01", DepositDay: 1, GoalID: "phone"}}, Expenses: []AssetPlanExpense{{Name: "Phone purchase", Amount: 1200000, Date: "2026-12-20", GoalID: "phone"}}}
}
func TestGoalPlanParallelSavingsAndExpense(t *testing.T) {
	s := phoneGoalPlan()
	m, err := s.Expand()
	require.NoError(t, err)
	assert.Equal(t, int64(1400000), m[0].CumulativePlanned)
	assert.Equal(t, int64(2800000), m[1].CumulativePlanned)
	assert.Equal(t, int64(3000000), m[2].CumulativePlanned)
	assert.Equal(t, int64(1200000), m[2].Expense)
	s.Goals[0].Amount = 9000000
	m, err = s.Expand()
	require.NoError(t, err)
	assert.Equal(t, int64(3000000), m[2].CumulativePlanned)
}
func TestGoalPlanCalendarAndValidation(t *testing.T) {
	s := MonthlyAssetPlanSchedule{Version: 2, StartMonth: "2024-01", EndMonth: "2024-03", FinalGoal: &AssetPlanGoal{Amount: 10000, Date: "2024-03-31"}, Stages: []AssetPlanStage{{Name: "Savings", Amount: 100, StartDate: "2024-01-31", EndDate: "2024-03-30", DepositDay: 31}}}
	events, err := s.Events()
	require.NoError(t, err)
	require.Len(t, events, 2)
	assert.Equal(t, "2024-02-29", events[1].Date)
	s.Stages[0].EndDate = ""
	s.Stages[0].Repeat = 4
	_, err = s.Expand()
	require.Error(t, err)
	s.Stages[0].Repeat = 3
	s.Stages[0].GoalID = "missing"
	_, err = s.Expand()
	require.Error(t, err)
	s.Stages[0].GoalID = ""
	s.Stages[0].Amount = assetPlanMaxSafe
	_, err = s.Expand()
	require.Error(t, err)
	s.Stages = nil
	s.Expenses = []AssetPlanExpense{{Name: "Spending existing assets", Amount: 200, Date: "2024-01-01"}}
	m, err := s.Expand()
	require.NoError(t, err)
	assert.Equal(t, int64(-200), m[2].CumulativePlanned)
}
