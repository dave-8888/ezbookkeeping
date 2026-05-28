package services

import (
	"time"

	"xorm.io/xorm"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/datastore"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
)

// UserLLMSettingsService represents user large language model settings service
type UserLLMSettingsService struct {
	ServiceUsingDB
}

// Initialize a user large language model settings service singleton instance
var (
	UserLLMSettings = &UserLLMSettingsService{
		ServiceUsingDB: ServiceUsingDB{
			container: datastore.Container,
		},
	}
)

// GetUserLLMSettingByUid returns user large language model setting model of user
func (s *UserLLMSettingsService) GetUserLLMSettingByUid(c core.Context, uid int64) (*models.UserLLMSetting, error) {
	if uid <= 0 {
		return nil, errs.ErrUserIdInvalid
	}

	userLLMSetting := &models.UserLLMSetting{}
	has, err := s.UserDB().NewSession(c).ID(uid).Get(userLLMSetting)

	if err != nil {
		return nil, err
	} else if !has {
		return nil, nil
	}

	return userLLMSetting, nil
}

// UpdateUserLLMSetting updates user large language model setting
func (s *UserLLMSettingsService) UpdateUserLLMSetting(c core.Context, userLLMSetting *models.UserLLMSetting) error {
	if userLLMSetting == nil || userLLMSetting.Uid <= 0 {
		return errs.ErrUserIdInvalid
	}

	userLLMSetting.UpdatedUnixTime = time.Now().Unix()

	return s.UserDB().DoTransaction(c, func(sess *xorm.Session) error {
		exists, err := sess.Cols("uid").Where("uid=?", userLLMSetting.Uid).Exist(&models.UserLLMSetting{})

		if err != nil {
			return err
		}

		updatedRows := int64(0)

		if !exists {
			updatedRows, err = sess.Insert(userLLMSetting)
		} else {
			updatedRows, err = sess.ID(userLLMSetting.Uid).AllCols().Update(userLLMSetting)
		}

		if err != nil {
			return err
		} else if updatedRows < 1 {
			log.Errorf(c, "[user_llm_settings.UpdateUserLLMSetting] failed to update user large language model setting")
			return errs.ErrDatabaseOperationFailed
		}

		return nil
	})
}
