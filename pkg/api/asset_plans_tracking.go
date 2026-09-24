package api

import (
	"encoding/json"
	"time"

	"github.com/mayswind/financially_free/pkg/errs"
	"github.com/mayswind/financially_free/pkg/models"
	"github.com/mayswind/financially_free/pkg/utils"
)

func accountIdStrings(ids []int64) []string {
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		result = append(result, utils.Int64ToString(id))
	}
	return result
}

func trackedPlanTotal(events []models.AssetPlanEvent, baseline, through string) (int64, error) {
	var total int64
	for _, event := range events {
		if event.Date > baseline && event.Date <= through {
			var err error
			total, err = monthlyDifference(total, -event.Amount)
			if err != nil {
				return 0, err
			}
		}
	}
	return total, nil
}

func applyBalanceTracking(plan *models.AssetPlan, monthly *models.MonthlyAssetPlanResponse, current int64) error {
	tracking := &models.AssetPlanActualTracking{}
	if plan.ActualTracking != "" {
		if err := json.Unmarshal([]byte(plan.ActualTracking), tracking); err != nil {
			return err
		}
	} else {
		tracking.Date, tracking.Amount = monthly.AsOfDate, current
	}
	if tracking.Snapshots == nil {
		tracking.Snapshots = map[string]int64{}
	}
	tracking.Snapshots[monthly.AsOfDate] = current
	if _, err := monthlyDifference(current, tracking.Amount); err != nil {
		return err
	}
	monthly.Tracking = tracking
	monthly.BaselineDate, monthly.BaselineAmount, monthly.ProjectionBase = tracking.Date, &tracking.Amount, &tracking.Amount
	events, err := monthly.MonthlyAssetPlanSchedule.Events()
	if err != nil {
		return err
	}
	monthly.TotalPlanned, err = trackedPlanTotal(events, tracking.Date, monthly.EndMonth+"-31")
	if err != nil {
		return err
	}
	monthly.PlannedToDate, err = trackedPlanTotal(events, tracking.Date, monthly.AsOfDate)
	if err != nil {
		return err
	}
	for i := range monthly.Months {
		row := &monthly.Months[i]
		first, _ := time.Parse("2006-01", row.Month)
		end := first.AddDate(0, 1, -1).Format("2006-01-02")
		before := first.AddDate(0, 0, -1).Format("2006-01-02")
		cumulative, err := trackedPlanTotal(events, tracking.Date, end)
		if err != nil {
			return err
		}
		previous, err := trackedPlanTotal(events, tracking.Date, before)
		if err != nil {
			return err
		}
		if _, err := monthlyDifference(tracking.Amount, -cumulative); err != nil {
			return err
		}
		row.CumulativePlanned = cumulative
		row.Planned = cumulative - previous
		row.Income, row.Expense = 0, 0
		for _, event := range events {
			if event.Date > tracking.Date && event.Date > before && event.Date <= end {
				if event.Amount >= 0 {
					row.Income += event.Amount
				} else {
					row.Expense -= event.Amount
				}
			}
		}
		through := end
		if through > monthly.AsOfDate {
			through = monthly.AsOfDate
		}
		if through < tracking.Date || row.Month > monthly.AsOfDate[:7] {
			continue
		}
		// Pick the latest observed balance in this month, bounded by today
		// and the tracking baseline. Never carry another month's value forward.
		latestDate := ""
		for date := range tracking.Snapshots {
			if date > before && date <= through && date >= tracking.Date && date > latestDate {
				latestDate = date
			}
		}
		if latestDate == "" {
			continue
		}
		through = latestDate
		balance := tracking.Snapshots[through]
		growth, err := monthlyDifference(balance, tracking.Amount)
		if err != nil {
			return err
		}
		planned, err := trackedPlanTotal(events, tracking.Date, through)
		if err != nil {
			return err
		}
		row.PlannedToDate = planned - previous
		difference, err := monthlyDifference(growth, planned)
		if err != nil {
			return err
		}
		row.CumulativeActual, row.Difference, row.ActualThrough = &growth, &difference, through
		opening, known := tracking.Snapshots[before]
		if before < tracking.Date {
			opening, known = tracking.Amount, true
		}
		if known {
			actual, err := monthlyDifference(balance, opening)
			if err != nil {
				return err
			}
			row.Actual = &actual
		}
	}
	encoded, err := json.Marshal(tracking)
	if err != nil {
		return err
	}
	plan.ActualTracking = string(encoded)
	return nil
}

func addTrackedDays(monthly *models.MonthlyAssetPlanResponse, month string, location *time.Location) error {
	first, err := time.ParseInLocation("2006-01", month, location)
	if err != nil || month < monthly.StartMonth || month > monthly.EndMonth {
		return errs.ErrNumberInvalid
	}
	events, err := monthly.MonthlyAssetPlanSchedule.Events()
	if err != nil {
		return err
	}
	tracking := monthly.Tracking
	for day := first; day.Month() == first.Month(); day = day.AddDate(0, 0, 1) {
		date := day.Format("2006-01-02")
		before := day.AddDate(0, 0, -1).Format("2006-01-02")
		row := models.AssetPlanMonth{Month: date}
		row.CumulativePlanned, err = trackedPlanTotal(events, tracking.Date, date)
		if err != nil {
			return err
		}
		for _, event := range events {
			if event.Date == date && date > tracking.Date {
				if event.Amount >= 0 {
					row.Income += event.Amount
				} else {
					row.Expense -= event.Amount
				}
			}
		}
		row.Planned = row.Income - row.Expense
		if balance, ok := tracking.Snapshots[date]; ok && date >= tracking.Date && date <= monthly.AsOfDate {
			growth, err := monthlyDifference(balance, tracking.Amount)
			if err != nil {
				return err
			}
			difference, err := monthlyDifference(growth, row.CumulativePlanned)
			if err != nil {
				return err
			}
			row.CumulativeActual, row.Difference, row.ActualThrough = &growth, &difference, date
			opening, known := tracking.Snapshots[before]
			if date == tracking.Date {
				opening, known = tracking.Amount, true
			}
			if known {
				actual, err := monthlyDifference(balance, opening)
				if err != nil {
					return err
				}
				row.Actual = &actual
			}
		}
		monthly.Days = append(monthly.Days, row)
	}
	return nil
}
