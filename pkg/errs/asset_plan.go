package errs

import "net/http"

// Error codes related to asset plans.
var (
	ErrAssetPlanAlreadyExists       = NewNormalError(NormalSubcategoryAssetPlan, 0, http.StatusBadRequest, "asset plan already exists")
	ErrAssetPlanNotFound            = NewNormalError(NormalSubcategoryAssetPlan, 1, http.StatusBadRequest, "asset plan not found")
	ErrAssetPlanTargetDateInvalid   = NewNormalError(NormalSubcategoryAssetPlan, 2, http.StatusBadRequest, "asset plan target date must be after today")
	ErrAssetPlanTargetAmountInvalid = NewNormalError(NormalSubcategoryAssetPlan, 3, http.StatusBadRequest, "asset plan target amount must be greater than start amount")
	ErrAssetPlanAccountScopeInvalid = NewNormalError(NormalSubcategoryAssetPlan, 4, http.StatusBadRequest, "asset plan account scope is invalid")
)
