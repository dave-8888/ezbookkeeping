package services

import (
	"time"

	"github.com/mayswind/financially_free/pkg/core"
	"github.com/mayswind/financially_free/pkg/datastore"
	"github.com/mayswind/financially_free/pkg/errs"
	"github.com/mayswind/financially_free/pkg/models"
)

// AssetPlanService represents the persistence service for user asset plans.
type AssetPlanService struct {
	ServiceUsingDB
}

// AssetPlans is the asset plan service singleton.
var AssetPlans = &AssetPlanService{
	ServiceUsingDB: ServiceUsingDB{
		container: datastore.Container,
	},
}

// GetAssetPlanByUid returns the current user's asset plan, or nil when absent.
func (s *AssetPlanService) GetAssetPlanByUid(c core.Context, uid int64) (*models.AssetPlan, error) {
	if uid <= 0 {
		return nil, errs.ErrUserIdInvalid
	}

	plan := &models.AssetPlan{}
	has, err := s.UserDataDB(uid).NewSession(c).ID(uid).Get(plan)

	if err != nil {
		return nil, err
	} else if !has {
		return nil, nil
	}

	return plan, nil
}

// CreateAssetPlan creates the single asset plan for a user.
func (s *AssetPlanService) CreateAssetPlan(c core.Context, plan *models.AssetPlan) error {
	if plan == nil || plan.Uid <= 0 {
		return errs.ErrUserIdInvalid
	}

	now := time.Now().Unix()
	plan.CreatedUnixTime = now
	plan.UpdatedUnixTime = now

	existing, err := s.GetAssetPlanByUid(c, plan.Uid)

	if err != nil {
		return err
	} else if existing != nil {
		return errs.ErrAssetPlanAlreadyExists
	}

	createdRows, err := s.UserDataDB(plan.Uid).NewSession(c).Insert(plan)

	if err != nil {
		existing, getErr := s.GetAssetPlanByUid(c, plan.Uid)

		if getErr == nil && existing != nil {
			return errs.ErrAssetPlanAlreadyExists
		}

		return err
	} else if createdRows < 1 {
		return errs.ErrDatabaseOperationFailed
	}

	return nil
}

// ModifyAssetPlan updates target fields while preserving the frozen start and scope.
func (s *AssetPlanService) ModifyAssetPlan(c core.Context, plan *models.AssetPlan) error {
	if plan == nil || plan.Uid <= 0 {
		return errs.ErrUserIdInvalid
	}

	plan.UpdatedUnixTime = time.Now().Unix()
	columns := []string{"target_amount", "target_date", "updated_unix_time"}
	if plan.MonthlySchedule != "" {
		columns = append(columns, "monthly_schedule", "start_date", "actual_tracking")
	}
	updatedRows, err := s.UserDataDB(plan.Uid).NewSession(c).
		ID(plan.Uid).
		Cols(columns...).
		Update(plan)

	if err != nil {
		return err
	} else if updatedRows < 1 {
		return errs.ErrAssetPlanNotFound
	}

	return nil
}

// DeleteAssetPlan deletes the user's asset plan so a new plan can be created.
func (s *AssetPlanService) DeleteAssetPlan(c core.Context, uid int64) error {
	if uid <= 0 {
		return errs.ErrUserIdInvalid
	}

	deletedRows, err := s.UserDataDB(uid).NewSession(c).ID(uid).Delete(&models.AssetPlan{})

	if err != nil {
		return err
	} else if deletedRows < 1 {
		return errs.ErrAssetPlanNotFound
	}

	return nil
}

// DeleteAssetPlanIfExists deletes the user's plan and succeeds when it is absent.
func (s *AssetPlanService) DeleteAssetPlanIfExists(c core.Context, uid int64) error {
	if uid <= 0 {
		return errs.ErrUserIdInvalid
	}

	_, err := s.UserDataDB(uid).NewSession(c).ID(uid).Delete(&models.AssetPlan{})
	return err
}

// SaveActualTracking updates observations without overwriting the user's plan.
func (s *AssetPlanService) SaveActualTracking(c core.Context, uid int64, tracking string) error {
	if uid <= 0 {
		return errs.ErrUserIdInvalid
	}
	rows, err := s.UserDataDB(uid).NewSession(c).ID(uid).Cols("actual_tracking").Update(&models.AssetPlan{ActualTracking: tracking})
	if err != nil {
		return err
	}
	if rows < 1 {
		return errs.ErrAssetPlanNotFound
	}
	return nil
}
