package models

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/mayswind/financially_free/pkg/utils"
)

const assetPlanDateFormat = "2006-01-02"

// Asset plan statuses.
const (
	ASSET_PLAN_STATUS_IN_PROGRESS  = "in_progress"
	ASSET_PLAN_STATUS_ACHIEVED     = "achieved"
	ASSET_PLAN_STATUS_NOT_ACHIEVED = "not_achieved"
)

// Asset plan comparison statuses.
const (
	ASSET_PLAN_COMPARISON_STATUS_AHEAD    = "ahead"
	ASSET_PLAN_COMPARISON_STATUS_BEHIND   = "behind"
	ASSET_PLAN_COMPARISON_STATUS_ON_TRACK = "on_track"
)

// AssetPlan represents the single net-worth plan stored for a user.
type AssetPlan struct {
	Uid                int64  `xorm:"PK"`
	TargetAmount       int64  `xorm:"NOT NULL"`
	TargetDate         int32  `xorm:"NOT NULL"`
	StartAmount        int64  `xorm:"NOT NULL"`
	StartDate          int32  `xorm:"NOT NULL"`
	Currency           string `xorm:"VARCHAR(3) NOT NULL"`
	ExcludedAccountIds string `xorm:"TEXT NOT NULL"`
	CreatedUnixTime    int64
	UpdatedUnixTime    int64
}

// AssetPlanCreateRequest represents all parameters for creating an asset plan.
type AssetPlanCreateRequest struct {
	TargetAmount       int64    `json:"targetAmount"`
	TargetDate         string   `json:"targetDate" binding:"required"`
	ExcludedAccountIds []string `json:"excludedAccountIds"`
}

// AssetPlanModifyRequest represents all parameters for modifying an asset plan.
type AssetPlanModifyRequest struct {
	TargetAmount int64  `json:"targetAmount"`
	TargetDate   string `json:"targetDate" binding:"required"`
}

// AssetPlanInfoResponse represents the current state of an asset plan.
type AssetPlanInfoResponse struct {
	TargetAmount             int64    `json:"targetAmount"`
	TargetDate               string   `json:"targetDate"`
	StartAmount              int64    `json:"startAmount"`
	StartDate                string   `json:"startDate"`
	Currency                 string   `json:"currency"`
	ExcludedAccountIds       []string `json:"excludedAccountIds"`
	CurrentActualNetWorth    int64    `json:"currentActualNetWorth"`
	TodayPlannedNetWorth     int64    `json:"todayPlannedNetWorth"`
	DifferenceAmount         int64    `json:"differenceAmount"`
	CompletionPercentage     string   `json:"completionPercentage"`
	Status                   string   `json:"status"`
	ComparisonStatus         string   `json:"comparisonStatus"`
	TargetDateActualNetWorth *int64   `json:"targetDateActualNetWorth,omitempty"`
}

// GetExcludedAccountIds returns the normalized excluded account ids.
func (p *AssetPlan) GetExcludedAccountIds() []int64 {
	if p.ExcludedAccountIds == "" {
		return []int64{}
	}

	accountIds, err := utils.StringArrayToInt64Array(strings.Split(p.ExcludedAccountIds, ","))

	if err != nil {
		return []int64{}
	}

	return accountIds
}

// SetExcludedAccountIds stores normalized excluded account ids.
func (p *AssetPlan) SetExcludedAccountIds(accountIds []int64) {
	values := make([]string, len(accountIds))

	for i := 0; i < len(accountIds); i++ {
		values[i] = utils.Int64ToString(accountIds[i])
	}

	p.ExcludedAccountIds = strings.Join(values, ",")
}

// ParseAssetPlanDate parses a strict YYYY-MM-DD date into its numeric representation.
func ParseAssetPlanDate(value string, location *time.Location) (int32, error) {
	if location == nil {
		location = time.UTC
	}

	parsed, err := time.ParseInLocation(assetPlanDateFormat, value, location)

	if err != nil || parsed.Format(assetPlanDateFormat) != value {
		return 0, fmt.Errorf("invalid asset plan date")
	}

	return AssetPlanNumericDate(parsed), nil
}

// AssetPlanNumericDate returns YYYYMMDD for a time value.
func AssetPlanNumericDate(value time.Time) int32 {
	return int32(value.Year()*10000 + int(value.Month())*100 + value.Day())
}

// AssetPlanDateToTime converts YYYYMMDD to the start of that natural day.
func AssetPlanDateToTime(value int32, location *time.Location) (time.Time, error) {
	if location == nil {
		location = time.UTC
	}

	year := int(value / 10000)
	month := time.Month((value % 10000) / 100)
	day := int(value % 100)
	parsed := time.Date(year, month, day, 0, 0, 0, 0, location)

	if AssetPlanNumericDate(parsed) != value {
		return time.Time{}, fmt.Errorf("invalid asset plan date")
	}

	return parsed, nil
}

// FormatAssetPlanDate formats YYYYMMDD as YYYY-MM-DD.
func FormatAssetPlanDate(value int32) string {
	return fmt.Sprintf("%04d-%02d-%02d", value/10000, (value%10000)/100, value%100)
}

// CalculateAssetPlanValue returns the planned net worth for a natural day.
func CalculateAssetPlanValue(plan *AssetPlan, currentDate int32, location *time.Location) (int64, error) {
	start, err := AssetPlanDateToTime(plan.StartDate, location)

	if err != nil {
		return 0, err
	}

	target, err := AssetPlanDateToTime(plan.TargetDate, location)

	if err != nil {
		return 0, err
	}

	current, err := AssetPlanDateToTime(currentDate, location)

	if err != nil {
		return 0, err
	}

	startCivilDate := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
	targetCivilDate := time.Date(target.Year(), target.Month(), target.Day(), 0, 0, 0, 0, time.UTC)
	currentCivilDate := time.Date(current.Year(), current.Month(), current.Day(), 0, 0, 0, 0, time.UTC)
	totalDays := int64(targetCivilDate.Sub(startCivilDate).Hours() / 24)

	if totalDays <= 0 {
		return 0, fmt.Errorf("asset plan target date must be after start date")
	}

	elapsedDays := int64(currentCivilDate.Sub(startCivilDate).Hours() / 24)

	if elapsedDays <= 0 {
		return plan.StartAmount, nil
	} else if elapsedDays >= totalDays {
		return plan.TargetAmount, nil
	}

	increment := new(big.Int).Sub(big.NewInt(plan.TargetAmount), big.NewInt(plan.StartAmount))
	increment.Mul(increment, big.NewInt(elapsedDays))
	increment.Quo(increment, big.NewInt(totalDays))
	increment.Add(increment, big.NewInt(plan.StartAmount))

	if !increment.IsInt64() {
		return 0, fmt.Errorf("asset plan value overflow")
	}

	return increment.Int64(), nil
}

// CalculateAssetPlanCompletionPercentage returns a clamped percentage with two decimal places.
func CalculateAssetPlanCompletionPercentage(startAmount int64, targetAmount int64, actualAmount int64) string {
	total := new(big.Int).Sub(big.NewInt(targetAmount), big.NewInt(startAmount))

	if total.Sign() <= 0 {
		return "0.00"
	}

	completed := new(big.Int).Sub(big.NewInt(actualAmount), big.NewInt(startAmount))

	if completed.Sign() <= 0 {
		return "0.00"
	} else if completed.Cmp(total) >= 0 {
		return "100.00"
	}

	percentage := new(big.Rat).SetFrac(completed, total)
	percentage.Mul(percentage, big.NewRat(100, 1))

	return percentage.FloatString(2)
}
