package api

import (
	"github.com/mayswind/financially_free/pkg/core"
	"github.com/mayswind/financially_free/pkg/errs"
)

func (a *SocialGiftsApi) PrivacyStatusHandler(c *core.WebContext) (any, *errs.Error) {
	value, err := a.socialGifts.GetPrivacy(c, c.GetCurrentUid())
	if err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}
	return map[string]bool{"configured": value != nil}, nil
}

func (a *SocialGiftsApi) PrivacyUnlockHandler(c *core.WebContext) (any, *errs.Error) {
	var req struct {
		Password string `json:"password" binding:"required,min=6,max=72"`
		Setup    bool   `json:"setup"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}
	token, err := a.socialGifts.UnlockPrivacy(c, c.GetCurrentUid(), req.Password, req.Setup)
	if err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}
	return map[string]string{"token": token}, nil
}

func (a *SocialGiftsApi) PrivacyLockHandler(c *core.WebContext) (any, *errs.Error) {
	if err := a.socialGifts.LockPrivacy(c, c.GetCurrentUid(), c.GetHeader("X-Social-Gift-Token")); err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}
	return true, nil
}
