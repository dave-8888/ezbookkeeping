package models

import (
	"fmt"
	"strings"
	"time"
)

// Goals annotate the plan; their amounts are never added to the cash flow.
type AssetPlanGoal struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Amount int64  `json:"amount"`
	Date   string `json:"date"`
}
type AssetPlanExpense struct {
	Name   string `json:"name"`
	Amount int64  `json:"amount"`
	Date   string `json:"date"`
	GoalID string `json:"goalId,omitempty"`
}
type AssetPlanEvent struct {
	Date   string
	Amount int64
	GoalID string
}

const assetPlanMaxSafe int64 = 9007199254740991

func (s MonthlyAssetPlanSchedule) Events() ([]AssetPlanEvent, error) {
	start, err := time.Parse("2006-01", s.StartMonth)
	if err != nil || start.Year() < 1970 {
		return nil, fmt.Errorf("invalid start month")
	}
	end, err := time.Parse("2006-01", s.EndMonth)
	if err != nil {
		return nil, fmt.Errorf("invalid end month")
	}
	count := (end.Year()-start.Year())*12 + int(end.Month()-start.Month()) + 1
	if count < 1 || count > 120 || len(s.Stages) > 120 || len(s.Goals) > 120 || len(s.Expenses) > 120 {
		return nil, fmt.Errorf("plan must cover 1 to 120 months")
	}
	last := end.AddDate(0, 1, -1).Format("2006-01-02")
	validDate := func(value string) bool {
		d, e := time.Parse("2006-01-02", value)
		return e == nil && d.Format("2006-01-02") == value && value >= s.StartMonth+"-01" && value <= last
	}
	validAmount := func(value int64) bool { return value > 0 && value <= assetPlanMaxSafe }
	validName := func(value string) bool { return strings.TrimSpace(value) != "" && len(value) <= 240 }
	if s.FinalGoal == nil || !validAmount(s.FinalGoal.Amount) || !validDate(s.FinalGoal.Date) || s.FinalGoal.Date != last {
		return nil, fmt.Errorf("final goal must be a positive net worth target at the plan end")
	}
	goals := make(map[string]bool)
	for _, g := range s.Goals {
		if g.ID == "" || len(g.ID) > 100 || goals[g.ID] || !validName(g.Name) || !validAmount(g.Amount) || !validDate(g.Date) {
			return nil, fmt.Errorf("invalid short term goal")
		}
		goals[g.ID] = true
	}
	events := make([]AssetPlanEvent, 0)
	var income, expense int64
	for _, stage := range s.Stages {
		if !validName(stage.Name) || stage.Amount < 0 || stage.Amount > assetPlanMaxSafe || !validDate(stage.StartDate) || stage.DepositDay < 1 || stage.DepositDay > 31 || (stage.GoalID != "" && !goals[stage.GoalID]) {
			return nil, fmt.Errorf("invalid savings schedule")
		}
		if stage.EndDate != "" {
			if !validDate(stage.EndDate) || stage.EndDate < stage.StartDate || stage.Repeat != 0 {
				return nil, fmt.Errorf("invalid savings end date")
			}
		} else if stage.Repeat < 1 || stage.Repeat > 120 {
			return nil, fmt.Errorf("invalid savings repeat")
		}
		first, _ := time.Parse("2006-01", stage.StartDate[:7])
		occurrences := 0
		for m := first; !m.After(end); m = m.AddDate(0, 1, 0) {
			day := stage.DepositDay
			maxDay := m.AddDate(0, 1, -1).Day()
			if day > maxDay {
				day = maxDay
			}
			date := m.AddDate(0, 0, day-1).Format("2006-01-02")
			if date < stage.StartDate {
				continue
			}
			if stage.EndDate != "" && date > stage.EndDate {
				break
			}
			if stage.EndDate == "" && occurrences == stage.Repeat {
				break
			}
			if stage.Amount > assetPlanMaxSafe-income {
				return nil, fmt.Errorf("plan amount overflow")
			}
			income += stage.Amount
			events = append(events, AssetPlanEvent{Date: date, Amount: stage.Amount, GoalID: stage.GoalID})
			occurrences++
		}
		if occurrences == 0 || (stage.EndDate == "" && occurrences != stage.Repeat) {
			return nil, fmt.Errorf("savings schedule exceeds plan range")
		}
	}
	for _, e := range s.Expenses {
		if !validName(e.Name) || !validAmount(e.Amount) || !validDate(e.Date) || (e.GoalID != "" && !goals[e.GoalID]) {
			return nil, fmt.Errorf("invalid expected expense")
		}
		if e.Amount > assetPlanMaxSafe-expense {
			return nil, fmt.Errorf("expense amount overflow")
		}
		expense += e.Amount
		events = append(events, AssetPlanEvent{Date: e.Date, Amount: -e.Amount, GoalID: e.GoalID})
	}
	return events, nil
}
func (s MonthlyAssetPlanSchedule) expandGoals() ([]AssetPlanMonth, error) {
	events, err := s.Events()
	if err != nil {
		return nil, err
	}
	start, _ := time.Parse("2006-01", s.StartMonth)
	end, _ := time.Parse("2006-01", s.EndMonth)
	months := make([]AssetPlanMonth, 0)
	var cumulative int64
	for m := start; !m.After(end); m = m.AddDate(0, 1, 0) {
		row := AssetPlanMonth{Month: m.Format("2006-01")}
		for _, e := range events {
			if e.Date[:7] == row.Month {
				if e.Amount >= 0 {
					row.Income += e.Amount
				} else {
					row.Expense -= e.Amount
				}
			}
		}
		row.Planned = row.Income - row.Expense
		cumulative += row.Planned
		row.CumulativePlanned = cumulative
		months = append(months, row)
	}
	return months, nil
}
