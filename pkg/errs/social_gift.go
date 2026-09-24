package errs

import "net/http"

var (
	ErrSocialGiftContactNotFound         = NewNormalError(NormalSubcategorySocialGift, 0, http.StatusBadRequest, "social gift contact not found")
	ErrSocialGiftContactInUse            = NewNormalError(NormalSubcategorySocialGift, 1, http.StatusBadRequest, "social gift contact is in use")
	ErrSocialGiftRecordNotFound          = NewNormalError(NormalSubcategorySocialGift, 2, http.StatusBadRequest, "social gift record not found")
	ErrSocialGiftDirectionInvalid        = NewNormalError(NormalSubcategorySocialGift, 3, http.StatusBadRequest, "social gift direction is invalid")
	ErrSocialGiftEventTypeInvalid        = NewNormalError(NormalSubcategorySocialGift, 4, http.StatusBadRequest, "social gift event type is invalid")
	ErrSocialGiftEventNameRequired       = NewNormalError(NormalSubcategorySocialGift, 5, http.StatusBadRequest, "social gift event name is required")
	ErrSocialGiftContactSelectionInvalid = NewNormalError(NormalSubcategorySocialGift, 6, http.StatusBadRequest, "social gift contact selection is invalid")
	ErrSocialGiftImportContactConflict   = NewNormalError(NormalSubcategorySocialGift, 7, http.StatusBadRequest, "social gift import contact key has conflicting data")
	ErrSocialGiftPrivacyLocked           = NewNormalError(NormalSubcategorySocialGift, 8, http.StatusForbidden, "social gift privacy is locked")
	ErrSocialGiftPrivacyPassword         = NewNormalError(NormalSubcategorySocialGift, 9, http.StatusBadRequest, "social gift privacy password is invalid")
	ErrSocialGiftPrivacyConfigured       = NewNormalError(NormalSubcategorySocialGift, 10, http.StatusBadRequest, "social gift privacy is already configured")
	ErrSocialGiftPrivacyRetry            = NewNormalError(NormalSubcategorySocialGift, 11, http.StatusTooManyRequests, "please wait before trying the privacy password again")
)
