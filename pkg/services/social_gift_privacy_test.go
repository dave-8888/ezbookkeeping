package services

import (
	"github.com/mayswind/financially_free/pkg/datastore"
	"github.com/mayswind/financially_free/pkg/errs"
	"github.com/mayswind/financially_free/pkg/models"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestSocialGiftPrivacyLifecycle(t *testing.T) {
	c := setupSocialGiftServiceTest(t)
	require.NoError(t, datastore.Container.UserDataStore.SyncStructs(new(models.SocialGiftPrivacy)))
	require.ErrorIs(t, SocialGifts.CheckPrivacy(c, 1, ""), errs.ErrSocialGiftPrivacyLocked)
	token, err := SocialGifts.UnlockPrivacy(c, 1, "secret-passphrase", true)
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.NoError(t, SocialGifts.CheckPrivacy(c, 1, token))
	require.Error(t, SocialGifts.CheckPrivacy(c, 2, token))
	value, err := SocialGifts.GetPrivacy(c, 1)
	require.NoError(t, err)
	require.NotEqual(t, "secret-passphrase", value.PasswordHash)
	require.NotEqual(t, token, value.TokenHash)
	_, err = SocialGifts.UnlockPrivacy(c, 1, "replacement", true)
	require.ErrorIs(t, err, errs.ErrSocialGiftPrivacyConfigured)
	require.NoError(t, SocialGifts.LockPrivacy(c, 1, token))
	require.Error(t, SocialGifts.CheckPrivacy(c, 1, token))
	_, err = SocialGifts.UnlockPrivacy(c, 1, "incorrect", false)
	require.ErrorIs(t, err, errs.ErrSocialGiftPrivacyPassword)
	_, err = SocialGifts.UnlockPrivacy(c, 1, "secret-passphrase", false)
	require.ErrorIs(t, err, errs.ErrSocialGiftPrivacyRetry)
	_, err = SocialGifts.UserDataDB(1).NewSession(c).ID(1).Cols("retry_after").Update(&models.SocialGiftPrivacy{RetryAfter: 0})
	require.NoError(t, err)
	newToken, err := SocialGifts.UnlockPrivacy(c, 1, "secret-passphrase", false)
	require.NoError(t, err)
	require.NoError(t, SocialGifts.CheckPrivacy(c, 1, newToken))
	// A delayed lock for the previous visit cannot revoke the new visit.
	require.NoError(t, SocialGifts.LockPrivacy(c, 1, token))
	require.NoError(t, SocialGifts.CheckPrivacy(c, 1, newToken))
	_, err = SocialGifts.UserDataDB(1).NewSession(c).ID(1).Cols("expires_at").Update(&models.SocialGiftPrivacy{ExpiresAt: time.Now().Unix() - 1})
	require.NoError(t, err)
	require.Error(t, SocialGifts.CheckPrivacy(c, 1, newToken))
}
