package api

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/mayswind/financially_free/pkg/core"
	"github.com/mayswind/financially_free/pkg/errs"
	"github.com/mayswind/financially_free/pkg/models"
	"github.com/mayswind/financially_free/pkg/utils"
)

func (a *AssetPlansApi) saveMonthlyAssetPlan(c *core.WebContext, existing *models.AssetPlan, schedule models.MonthlyAssetPlanSchedule, excluded []string, location *time.Location) (any, *errs.Error) {
	start, end := schedule.StartMonth, schedule.EndMonth
	months, err := schedule.Expand()
	if err != nil {
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}
	var plan models.AssetPlan
	if existing != nil {
		plan = *existing
	} else {
		plan.Uid = c.GetCurrentUid()
		user, err := a.users.GetUserById(c, plan.Uid)
		if err != nil {
			return nil, errs.Or(err, errs.ErrOperationFailed)
		}
		plan.Currency = user.DefaultCurrency
		ids, err := a.netWorth.ResolveExcludedAccountIds(c, plan.Uid, excluded)
		if err != nil {
			return nil, errs.Or(err, errs.ErrAssetPlanAccountScopeInvalid)
		}
		plan.SetExcludedAccountIds(ids)
	}
	data, err := json.Marshal(schedule)
	if err != nil {
		return nil, errs.ErrOperationFailed
	}
	plan.MonthlySchedule = string(data)
	first, _ := time.ParseInLocation("2006-01", start, location)
	last, _ := time.ParseInLocation("2006-01", end, location)
	plan.StartDate = models.AssetPlanNumericDate(first)
	plan.TargetDate = models.AssetPlanNumericDate(last.AddDate(0, 1, -1))
	plan.TargetAmount = months[len(months)-1].CumulativePlanned
	if schedule.Version == 2 {
		plan.TargetAmount = schedule.FinalGoal.Amount
	}
	// Calculate before persisting so a failed ledger read does not save an unseen change.
	response, err := a.buildMonthlyAssetPlanResponse(c, &plan, location)
	if err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}
	if existing == nil {
		err = a.assetPlans.CreateAssetPlan(c, &plan)
	} else {
		err = a.assetPlans.ModifyAssetPlan(c, &plan)
	}
	if err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}
	return response, nil
}

func monthlyDifference(first, second int64) (int64, error) {
	value := new(big.Int).Sub(big.NewInt(first), big.NewInt(second))
	if !value.IsInt64() || value.Cmp(big.NewInt(9007199254740991)) > 0 || value.Cmp(big.NewInt(-9007199254740991)) < 0 {
		return 0, errs.ErrNumberInvalid
	}
	return value.Int64(), nil
}

func (a *AssetPlansApi) buildMonthlyAssetPlanResponse(c core.Context, plan *models.AssetPlan, location *time.Location) (*models.AssetPlanInfoResponse, error) {
	var schedule models.MonthlyAssetPlanSchedule
	if err := json.Unmarshal([]byte(plan.MonthlySchedule), &schedule); err != nil {
		return nil, err
	}
	months, err := schedule.Expand()
	if err != nil {
		return nil, err
	}
	today := a.now().In(location).Format("2006-01-02")
	start, _ := time.ParseInLocation("2006-01", schedule.StartMonth, location)
	baselineDate := start.AddDate(0, 0, -1)
	monthly := &models.MonthlyAssetPlanResponse{MonthlyAssetPlanSchedule: schedule, Months: months, BaselineDate: baselineDate.Format("2006-01-02"), AsOfDate: today, TotalPlanned: months[len(months)-1].CumulativePlanned}
	if schedule.Version == 2 {
		current, err := a.netWorth.CalculateCurrentNetWorth(c, plan.Uid, plan.Currency, plan.GetExcludedAccountIds())
		if err != nil {
			return nil, err
		}
		monthly.CurrentNetWorth = &current
		if err := applyBalanceTracking(plan, monthly, current); err != nil {
			return nil, err
		}
		return &models.AssetPlanInfoResponse{Monthly: monthly, Currency: plan.Currency, ExcludedAccountIds: accountIdStrings(plan.GetExcludedAccountIds()), StartDate: models.FormatAssetPlanDate(plan.StartDate), TargetDate: models.FormatAssetPlanDate(plan.TargetDate), TargetAmount: schedule.FinalGoal.Amount}, nil
	}
	// Future plans have no actual data or estimated baseline.
	if today >= start.Format("2006-01-02") {
		baseline, err := a.netWorth.CalculateNetWorth(c, plan.Uid, models.AssetPlanNumericDate(baselineDate), location, plan.Currency, plan.GetExcludedAccountIds())
		if err != nil {
			return nil, err
		}
		monthly.BaselineAmount = &baseline
		previous := baseline
		for i := range months {
			month := &months[i]
			if month.Month > today[:7] {
				break
			}
			first, _ := time.ParseInLocation("2006-01", month.Month, location)
			through := first.AddDate(0, 1, -1).Format("2006-01-02")
			if through > today {
				through = today
			}
			date, _ := models.ParseAssetPlanDate(through, location)
			balance, err := a.netWorth.CalculateNetWorth(c, plan.Uid, date, location, plan.Currency, plan.GetExcludedAccountIds())
			if err != nil {
				return nil, err
			}
			actual, err := monthlyDifference(balance, previous)
			if err != nil {
				return nil, err
			}
			cumulative, err := monthlyDifference(balance, baseline)
			if err != nil {
				return nil, err
			}
			difference, err := monthlyDifference(cumulative, month.CumulativePlanned)
			if err != nil {
				return nil, err
			}
			month.Actual, month.CumulativeActual, month.Difference = &actual, &cumulative, &difference
			month.ActualThrough = through
			previous = balance
		}
	}
	monthly.ProjectionBase = monthly.BaselineAmount
	if schedule.Version == 2 && monthly.ProjectionBase == nil {
		monthly.ProjectionBase = monthly.CurrentNetWorth
	}
	if monthly.ProjectionBase != nil {
		for _, month := range months {
			if _, err := monthlyDifference(*monthly.ProjectionBase, -month.CumulativePlanned); err != nil {
				return nil, err
			}
		}
	}
	ids := make([]string, 0)
	for _, id := range plan.GetExcludedAccountIds() {
		ids = append(ids, utils.Int64ToString(id))
	}
	targetAmount := monthly.TotalPlanned
	if schedule.Version == 2 {
		targetAmount = schedule.FinalGoal.Amount
	}
	return &models.AssetPlanInfoResponse{Monthly: monthly, Currency: plan.Currency, ExcludedAccountIds: ids, StartDate: models.FormatAssetPlanDate(plan.StartDate), TargetDate: models.FormatAssetPlanDate(plan.TargetDate), TargetAmount: targetAmount}, nil
}

// Daily detail is bounded to one requested month; actual values use the same ledger provider.
func (a *AssetPlansApi) addMonthlyPlanDays(c core.Context, plan *models.AssetPlan, monthly *models.MonthlyAssetPlanResponse, month string, location *time.Location) error {
	if monthly.Tracking != nil {
		return addTrackedDays(monthly, month, location)
	}
	first, err := time.ParseInLocation("2006-01", month, location)
	if err != nil || month < monthly.StartMonth || month > monthly.EndMonth {
		return errs.ErrNumberInvalid
	}
	events := []models.AssetPlanEvent{}
	if monthly.Version == 2 {
		events, err = monthly.MonthlyAssetPlanSchedule.Events()
		if err != nil {
			return err
		}
	} else {
		for _, m := range monthly.Months {
			date, _ := time.ParseInLocation("2006-01", m.Month, location)
			events = append(events, models.AssetPlanEvent{Date: date.AddDate(0, 1, -1).Format("2006-01-02"), Amount: m.Planned})
		}
	}
	var total int64
	for _, e := range events {
		if e.Date < month+"-01" {
			total += e.Amount
		}
	}
	var previous int64
	if monthly.BaselineAmount != nil {
		previous = *monthly.BaselineAmount
		for _, m := range monthly.Months {
			if m.Month < month && m.CumulativeActual != nil {
				previous = *monthly.BaselineAmount + *m.CumulativeActual
			}
		}
	}
	for day := first; day.Month() == first.Month(); day = day.AddDate(0, 0, 1) {
		date := day.Format("2006-01-02")
		row := models.AssetPlanMonth{Month: date}
		for _, e := range events {
			if e.Date == date {
				if e.Amount >= 0 {
					row.Income += e.Amount
				} else {
					row.Expense -= e.Amount
				}
			}
		}
		row.Planned = row.Income - row.Expense
		total += row.Planned
		row.CumulativePlanned = total
		if monthly.ProjectionBase != nil {
			if _, err := monthlyDifference(*monthly.ProjectionBase, -total); err != nil {
				return err
			}
		}
		if date <= monthly.AsOfDate && monthly.BaselineAmount != nil {
			balance, err := a.netWorth.CalculateNetWorth(c, plan.Uid, models.AssetPlanNumericDate(day), location, plan.Currency, plan.GetExcludedAccountIds())
			if err != nil {
				return err
			}
			actual, err := monthlyDifference(balance, previous)
			if err != nil {
				return err
			}
			cumulative, err := monthlyDifference(balance, *monthly.BaselineAmount)
			if err != nil {
				return err
			}
			difference, err := monthlyDifference(cumulative, total)
			if err != nil {
				return err
			}
			row.Actual = &actual
			row.CumulativeActual = &cumulative
			row.Difference = &difference
			row.ActualThrough = date
			previous = balance
		}
		monthly.Days = append(monthly.Days, row)
	}
	return nil
}
