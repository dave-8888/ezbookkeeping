package models

import (
	"fmt"
	"time"
)

// AssetPlanStage repeats an amount for consecutive months, in stored order.
type AssetPlanStage struct {
	Amount     int64  `json:"amount"`
	Repeat     int    `json:"repeat"`
	Name       string `json:"name,omitempty"`
	StartDate  string `json:"startDate,omitempty"`
	EndDate    string `json:"endDate,omitempty"`
	DepositDay int    `json:"depositDay,omitempty"`
	GoalID     string `json:"goalId,omitempty"`
}

type MonthlyAssetPlanSchedule struct {
	Version    int                `json:"version,omitempty"`
	FinalGoal  *AssetPlanGoal     `json:"finalGoal,omitempty"`
	Goals      []AssetPlanGoal    `json:"goals,omitempty"`
	Expenses   []AssetPlanExpense `json:"expenses,omitempty"`
	StartMonth string             `json:"startMonth"`
	EndMonth   string             `json:"endMonth"`
	Stages     []AssetPlanStage   `json:"stages"`
}

type AssetPlanMonth struct {
	PlannedToDate     int64  `json:"plannedToDate"`
	Income            int64  `json:"income"`
	Expense           int64  `json:"expense"`
	Month             string `json:"month"`
	Planned           int64  `json:"planned"`
	CumulativePlanned int64  `json:"cumulativePlanned"`
	Actual            *int64 `json:"actual"`
	CumulativeActual  *int64 `json:"cumulativeActual"`
	Difference        *int64 `json:"difference"`
	ActualThrough     string `json:"actualThrough,omitempty"`
}

// AssetPlanActualTracking stores observed balances, never inferred historical values.
type AssetPlanActualTracking struct {
	Date      string           `json:"date"`
	Amount    int64            `json:"amount"`
	Snapshots map[string]int64 `json:"snapshots"`
}

type MonthlyAssetPlanResponse struct {
	Tracking      *AssetPlanActualTracking `json:"tracking,omitempty"`
	PlannedToDate int64                    `json:"plannedToDate"`

	MonthlyAssetPlanSchedule
	CurrentNetWorth *int64           `json:"currentNetWorth"`
	Months          []AssetPlanMonth `json:"months"`
	ProjectionBase  *int64           `json:"projectionBase"`
	BaselineDate    string           `json:"baselineDate"`
	BaselineAmount  *int64           `json:"baselineAmount"`
	AsOfDate        string           `json:"asOfDate"`
	Days            []AssetPlanMonth `json:"days,omitempty"`
	TotalPlanned    int64            `json:"totalPlanned"`
}

// Expand validates an inclusive monthly range and an exact stage coverage.
// Safe integer bounds keep monetary values exact in both Go and JavaScript.
func (s MonthlyAssetPlanSchedule) Expand() ([]AssetPlanMonth, error) {
	if s.Version == 2 {
		return s.expandGoals()
	}
	if s.Version != 0 {
		return nil, fmt.Errorf("unsupported plan version")
	}
	start, err := time.Parse("2006-01", s.StartMonth)
	if err != nil || start.Format("2006-01") != s.StartMonth || start.Year() < 1970 {
		return nil, fmt.Errorf("invalid start month")
	}
	end, err := time.Parse("2006-01", s.EndMonth)
	if err != nil || end.Format("2006-01") != s.EndMonth {
		return nil, fmt.Errorf("invalid end month")
	}
	count := (end.Year()-start.Year())*12 + int(end.Month()-start.Month()) + 1
	if count < 1 || count > 120 || len(s.Stages) < 1 || len(s.Stages) > count {
		return nil, fmt.Errorf("plan must cover 1 to 120 months")
	}
	months := make([]AssetPlanMonth, 0, count)
	var total int64
	const maxSafe = int64(9007199254740991)
	for _, stage := range s.Stages {
		if stage.Amount < 0 || stage.Amount > maxSafe || stage.Repeat < 1 || stage.Repeat > count-len(months) {
			return nil, fmt.Errorf("invalid monthly stage")
		}
		for i := 0; i < stage.Repeat; i++ {
			if stage.Amount > maxSafe-total {
				return nil, fmt.Errorf("plan amount overflow")
			}
			total += stage.Amount
			months = append(months, AssetPlanMonth{Month: start.AddDate(0, len(months), 0).Format("2006-01"), Planned: stage.Amount, CumulativePlanned: total})
		}
	}
	if len(months) != count || total == 0 {
		return nil, fmt.Errorf("stages must cover the range and total must be positive")
	}
	return months, nil
}
