package api

import (
	"math/big"
	"sort"
	"strings"
	"time"

	"github.com/mayswind/financially_free/pkg/core"
	"github.com/mayswind/financially_free/pkg/errs"
	"github.com/mayswind/financially_free/pkg/exchangerates"
	"github.com/mayswind/financially_free/pkg/log"
	"github.com/mayswind/financially_free/pkg/models"
	"github.com/mayswind/financially_free/pkg/services"
	"github.com/mayswind/financially_free/pkg/settings"
	"github.com/mayswind/financially_free/pkg/utils"
)

type assetPlanStore interface {
	GetAssetPlanByUid(c core.Context, uid int64) (*models.AssetPlan, error)
	CreateAssetPlan(c core.Context, plan *models.AssetPlan) error
	ModifyAssetPlan(c core.Context, plan *models.AssetPlan) error
	DeleteAssetPlan(c core.Context, uid int64) error
}

type assetPlanUserProvider interface {
	GetUserById(c core.Context, uid int64) (*models.User, error)
}

type assetPlanNetWorthProvider interface {
	ResolveExcludedAccountIds(c core.Context, uid int64, accountIds []string) ([]int64, error)
	CalculateNetWorth(c core.Context, uid int64, date int32, location *time.Location, currency string, excludedAccountIds []int64) (int64, error)
}

type latestExchangeRatesProvider interface {
	GetLatestExchangeRates(c core.Context, uid int64, currentConfig *settings.Config) (*models.LatestExchangeRateResponse, error)
}

type defaultAssetPlanNetWorthProvider struct {
	accounts      *services.AccountService
	transactions  *services.TransactionService
	exchangeRates latestExchangeRatesProvider
	config        *settings.ConfigContainer
}

// AssetPlansApi represents the asset plan API.
type AssetPlansApi struct {
	assetPlans assetPlanStore
	users      assetPlanUserProvider
	netWorth   assetPlanNetWorthProvider
	now        func() time.Time
}

// AssetPlans is the asset plan API singleton.
var AssetPlans = &AssetPlansApi{
	assetPlans: services.AssetPlans,
	users:      services.Users,
	netWorth: &defaultAssetPlanNetWorthProvider{
		accounts:      services.Accounts,
		transactions:  services.Transactions,
		exchangeRates: exchangerates.Container,
		config:        settings.Container,
	},
	now: time.Now,
}

// AssetPlanGetHandler returns the current user's asset plan.
func (a *AssetPlansApi) AssetPlanGetHandler(c *core.WebContext) (any, *errs.Error) {
	uid := c.GetCurrentUid()
	plan, err := a.assetPlans.GetAssetPlanByUid(c, uid)

	if err != nil {
		log.Errorf(c, "[asset_plans.AssetPlanGetHandler] failed to get asset plan for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	} else if plan == nil {
		return nil, nil
	}

	location, finalErr := getAssetPlanClientTimezone(c)

	if finalErr != nil {
		return nil, finalErr
	}

	response, err := a.buildAssetPlanResponse(c, plan, location, nil)

	if err != nil {
		log.Errorf(c, "[asset_plans.AssetPlanGetHandler] failed to calculate asset plan for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	return response, nil
}

// AssetPlanCreateHandler creates the current user's single asset plan.
func (a *AssetPlansApi) AssetPlanCreateHandler(c *core.WebContext) (any, *errs.Error) {
	var req models.AssetPlanCreateRequest
	err := c.ShouldBindJSON(&req)

	if err != nil {
		log.Warnf(c, "[asset_plans.AssetPlanCreateHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	location, finalErr := getAssetPlanClientTimezone(c)

	if finalErr != nil {
		return nil, finalErr
	}

	uid := c.GetCurrentUid()
	existing, err := a.assetPlans.GetAssetPlanByUid(c, uid)

	if err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	} else if existing != nil {
		return nil, errs.ErrAssetPlanAlreadyExists
	}

	targetDate, err := models.ParseAssetPlanDate(req.TargetDate, location)

	if err != nil {
		return nil, errs.ErrAssetPlanTargetDateInvalid
	}

	today := models.AssetPlanNumericDate(a.now().In(location))

	if targetDate <= today {
		return nil, errs.ErrAssetPlanTargetDateInvalid
	}

	user, err := a.users.GetUserById(c, uid)

	if err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	excludedAccountIds, err := a.netWorth.ResolveExcludedAccountIds(c, uid, req.ExcludedAccountIds)

	if err != nil {
		return nil, errs.Or(err, errs.ErrAssetPlanAccountScopeInvalid)
	}

	startAmount, err := a.netWorth.CalculateNetWorth(c, uid, today, location, user.DefaultCurrency, excludedAccountIds)

	if err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	if req.TargetAmount <= startAmount {
		return nil, errs.ErrAssetPlanTargetAmountInvalid
	}

	plan := &models.AssetPlan{
		Uid:          uid,
		TargetAmount: req.TargetAmount,
		TargetDate:   targetDate,
		StartAmount:  startAmount,
		StartDate:    today,
		Currency:     user.DefaultCurrency,
	}
	plan.SetExcludedAccountIds(excludedAccountIds)

	err = a.assetPlans.CreateAssetPlan(c, plan)

	if err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	response, err := a.buildAssetPlanResponse(c, plan, location, &startAmount)

	if err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	log.Infof(c, "[asset_plans.AssetPlanCreateHandler] user \"uid:%d\" has created an asset plan", uid)
	return response, nil
}

// AssetPlanModifyHandler updates target fields while preserving the original start and scope.
func (a *AssetPlansApi) AssetPlanModifyHandler(c *core.WebContext) (any, *errs.Error) {
	var req models.AssetPlanModifyRequest
	err := c.ShouldBindJSON(&req)

	if err != nil {
		log.Warnf(c, "[asset_plans.AssetPlanModifyHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	location, finalErr := getAssetPlanClientTimezone(c)

	if finalErr != nil {
		return nil, finalErr
	}

	uid := c.GetCurrentUid()
	plan, err := a.assetPlans.GetAssetPlanByUid(c, uid)

	if err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	} else if plan == nil {
		return nil, errs.ErrAssetPlanNotFound
	}

	targetDate, err := models.ParseAssetPlanDate(req.TargetDate, location)

	if err != nil {
		return nil, errs.ErrAssetPlanTargetDateInvalid
	}

	today := models.AssetPlanNumericDate(a.now().In(location))

	if targetDate <= today {
		return nil, errs.ErrAssetPlanTargetDateInvalid
	} else if req.TargetAmount <= plan.StartAmount {
		return nil, errs.ErrAssetPlanTargetAmountInvalid
	}

	plan.TargetAmount = req.TargetAmount
	plan.TargetDate = targetDate
	err = a.assetPlans.ModifyAssetPlan(c, plan)

	if err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	response, err := a.buildAssetPlanResponse(c, plan, location, nil)

	if err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	log.Infof(c, "[asset_plans.AssetPlanModifyHandler] user \"uid:%d\" has modified the asset plan", uid)
	return response, nil
}

// AssetPlanDeleteHandler deletes the current user's asset plan.
func (a *AssetPlansApi) AssetPlanDeleteHandler(c *core.WebContext) (any, *errs.Error) {
	uid := c.GetCurrentUid()
	err := a.assetPlans.DeleteAssetPlan(c, uid)

	if err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	log.Infof(c, "[asset_plans.AssetPlanDeleteHandler] user \"uid:%d\" has deleted the asset plan", uid)
	return true, nil
}

func (a *AssetPlansApi) buildAssetPlanResponse(c core.Context, plan *models.AssetPlan, location *time.Location, currentActual *int64) (*models.AssetPlanInfoResponse, error) {
	today := models.AssetPlanNumericDate(a.now().In(location))
	actual := int64(0)
	var err error

	if currentActual != nil {
		actual = *currentActual
	} else {
		actual, err = a.netWorth.CalculateNetWorth(c, plan.Uid, today, location, plan.Currency, plan.GetExcludedAccountIds())

		if err != nil {
			return nil, err
		}
	}

	planned, err := models.CalculateAssetPlanValue(plan, today, location)

	if err != nil {
		return nil, err
	}

	difference := actual - planned
	comparisonStatus := models.ASSET_PLAN_COMPARISON_STATUS_ON_TRACK

	if difference > 0 {
		comparisonStatus = models.ASSET_PLAN_COMPARISON_STATUS_AHEAD
	} else if difference < 0 {
		comparisonStatus = models.ASSET_PLAN_COMPARISON_STATUS_BEHIND
	}

	status := models.ASSET_PLAN_STATUS_IN_PROGRESS
	var targetDateActual *int64

	if today >= plan.TargetDate {
		targetActual := actual

		if today != plan.TargetDate {
			targetActual, err = a.netWorth.CalculateNetWorth(c, plan.Uid, plan.TargetDate, location, plan.Currency, plan.GetExcludedAccountIds())

			if err != nil {
				return nil, err
			}
		}

		targetDateActual = &targetActual

		if targetActual >= plan.TargetAmount {
			status = models.ASSET_PLAN_STATUS_ACHIEVED
		} else {
			status = models.ASSET_PLAN_STATUS_NOT_ACHIEVED
		}
	}

	excludedAccountIds := plan.GetExcludedAccountIds()
	excludedAccountIdStrings := make([]string, len(excludedAccountIds))

	for i := 0; i < len(excludedAccountIds); i++ {
		excludedAccountIdStrings[i] = utils.Int64ToString(excludedAccountIds[i])
	}

	return &models.AssetPlanInfoResponse{
		TargetAmount:             plan.TargetAmount,
		TargetDate:               models.FormatAssetPlanDate(plan.TargetDate),
		StartAmount:              plan.StartAmount,
		StartDate:                models.FormatAssetPlanDate(plan.StartDate),
		Currency:                 plan.Currency,
		ExcludedAccountIds:       excludedAccountIdStrings,
		CurrentActualNetWorth:    actual,
		TodayPlannedNetWorth:     planned,
		DifferenceAmount:         difference,
		CompletionPercentage:     models.CalculateAssetPlanCompletionPercentage(plan.StartAmount, plan.TargetAmount, actual),
		Status:                   status,
		ComparisonStatus:         comparisonStatus,
		TargetDateActualNetWorth: targetDateActual,
	}, nil
}

func getAssetPlanClientTimezone(c *core.WebContext) (*time.Location, *errs.Error) {
	location, err := c.GetClientTimezone()

	if err != nil {
		log.Warnf(c, "[asset_plans.getAssetPlanClientTimezone] cannot get client timezone, because %s", err.Error())
		return nil, errs.ErrClientTimezoneOffsetInvalid
	}

	return location, nil
}

func (p *defaultAssetPlanNetWorthProvider) ResolveExcludedAccountIds(c core.Context, uid int64, accountIdStrings []string) ([]int64, error) {
	if len(accountIdStrings) < 1 {
		return []int64{}, nil
	}

	accountIds, err := utils.StringArrayToInt64Array(accountIdStrings)

	if err != nil {
		return nil, errs.ErrAssetPlanAccountScopeInvalid
	}

	accountIds = utils.ToUniqueInt64Slice(accountIds)

	for i := 0; i < len(accountIds); i++ {
		if accountIds[i] <= 0 {
			return nil, errs.ErrAssetPlanAccountScopeInvalid
		}
	}

	accountMap, err := p.accounts.GetAccountsByAccountIds(c, uid, accountIds)

	if err != nil {
		return nil, err
	} else if len(accountMap) != len(accountIds) {
		return nil, errs.ErrAssetPlanAccountScopeInvalid
	}

	values := make([]string, len(accountIds))

	for i := 0; i < len(accountIds); i++ {
		values[i] = utils.Int64ToString(accountIds[i])
	}

	resolved, err := p.accounts.GetAccountOrSubAccountIds(c, strings.Join(values, ","), uid)

	if err != nil {
		return nil, err
	}

	resolved = utils.ToUniqueInt64Slice(resolved)
	sort.Slice(resolved, func(i, j int) bool {
		return resolved[i] < resolved[j]
	})

	return resolved, nil
}

func (p *defaultAssetPlanNetWorthProvider) CalculateNetWorth(c core.Context, uid int64, date int32, location *time.Location, currency string, excludedAccountIds []int64) (int64, error) {
	day, err := models.AssetPlanDateToTime(date, location)

	if err != nil {
		return 0, err
	}

	minTransactionTime := utils.GetMinTransactionTimeFromUnixTime(day.Unix())
	maxTransactionTime := utils.GetMaxTransactionTimeFromUnixTime(day.AddDate(0, 0, 1).Add(-time.Second).Unix())
	dailyBalances, err := p.transactions.GetAllAccountsDailyOpeningAndClosingBalance(c, uid, maxTransactionTime, minTransactionTime, location)

	if err != nil {
		return 0, err
	}

	balances := dailyBalances[date]

	if len(balances) < 1 {
		return 0, nil
	}

	accounts, err := p.accounts.GetAllAccountsByUid(c, uid)

	if err != nil {
		return 0, err
	}

	accountMap := p.accounts.GetAccountMapByList(accounts)
	excludedMap := make(map[int64]bool, len(excludedAccountIds))

	for i := 0; i < len(excludedAccountIds); i++ {
		excludedMap[excludedAccountIds[i]] = true
	}

	needsExchangeRates := false

	for _, balance := range balances {
		account := accountMap[balance.AccountId]

		if account != nil &&
			!excludedMap[account.AccountId] &&
			(account.Category.IsAsset() || account.Category.IsLiability()) &&
			balance.AccountClosingBalance != 0 &&
			account.Currency != currency {
			needsExchangeRates = true
			break
		}
	}

	rateMap := make(map[string]*big.Rat)

	if needsExchangeRates {
		exchangeRateResponse, err := p.exchangeRates.GetLatestExchangeRates(c, uid, p.config.GetCurrentConfig())

		if err != nil {
			return 0, err
		}

		rateMap = make(map[string]*big.Rat, len(exchangeRateResponse.ExchangeRates))

		for i := 0; i < len(exchangeRateResponse.ExchangeRates); i++ {
			rate := exchangeRateResponse.ExchangeRates[i]
			rateValue, ok := new(big.Rat).SetString(rate.Rate)

			if ok && rateValue.Sign() > 0 {
				rateMap[rate.Currency] = rateValue
			}
		}
	}

	total := big.NewInt(0)

	for _, balance := range balances {
		account := accountMap[balance.AccountId]

		if account == nil || excludedMap[account.AccountId] || (!account.Category.IsAsset() && !account.Category.IsLiability()) {
			continue
		}

		converted := balance.AccountClosingBalance
		ok := true

		if account.Currency != currency {
			converted, ok = convertAssetPlanAmount(balance.AccountClosingBalance, rateMap[account.Currency], rateMap[currency])
		}

		if !ok {
			continue
		}

		// Account balances already carry their sign. This matches the existing
		// asset-trends net-worth calculation for both assets and liabilities.
		total.Add(total, big.NewInt(converted))
	}

	if !total.IsInt64() {
		return 0, errs.ErrNumberInvalid
	}

	return total.Int64(), nil
}

func convertAssetPlanAmount(amount int64, fromRate *big.Rat, toRate *big.Rat) (int64, bool) {
	if amount == 0 {
		return 0, true
	} else if fromRate == nil || toRate == nil || fromRate.Sign() <= 0 || toRate.Sign() <= 0 {
		return 0, false
	}

	value := new(big.Rat).SetInt64(amount)
	value.Mul(value, toRate)
	value.Quo(value, fromRate)
	quotient := new(big.Int).Quo(value.Num(), value.Denom())

	if !quotient.IsInt64() {
		return 0, false
	}

	return quotient.Int64(), true
}
