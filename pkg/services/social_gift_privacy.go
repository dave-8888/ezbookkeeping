package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"github.com/mayswind/financially_free/pkg/core"
	"github.com/mayswind/financially_free/pkg/errs"
	"github.com/mayswind/financially_free/pkg/models"
	"golang.org/x/crypto/bcrypt"
	"time"
	"unicode/utf8"
)

func (s *SocialGiftService) GetPrivacy(c core.Context, uid int64) (*models.SocialGiftPrivacy, error) {
	value := &models.SocialGiftPrivacy{}
	has, err := s.UserDataDB(uid).NewSession(c).ID(uid).Get(value)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, nil
	}
	return value, nil
}

func (s *SocialGiftService) UnlockPrivacy(c core.Context, uid int64, password string, setup bool) (string, error) {
	if utf8.RuneCountInString(password) < 6 || len(password) > 72 {
		return "", errs.ErrSocialGiftPrivacyPassword
	}
	value, err := s.GetPrivacy(c, uid)
	if err != nil {
		return "", err
	}
	if setup {
		if value != nil {
			return "", errs.ErrSocialGiftPrivacyConfigured
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return "", err
		}
		value = &models.SocialGiftPrivacy{Uid: uid, PasswordHash: string(hash)}
		if _, err = s.UserDataDB(uid).NewSession(c).Insert(value); err != nil {
			return "", err
		}
	} else {
		if value == nil {
			return "", errs.ErrSocialGiftPrivacyLocked
		}
		now := time.Now().Unix()
		if value.RetryAfter > now {
			return "", errs.ErrSocialGiftPrivacyRetry
		}
		// Reserve a short retry window atomically, including across server instances.
		rows, err := s.UserDataDB(uid).NewSession(c).Where("uid=? AND retry_after<=?", uid, now).Cols("retry_after").Update(&models.SocialGiftPrivacy{RetryAfter: now + 2})
		if err != nil {
			return "", err
		}
		if rows != 1 {
			return "", errs.ErrSocialGiftPrivacyRetry
		}
		if bcrypt.CompareHashAndPassword([]byte(value.PasswordHash), []byte(password)) != nil {
			return "", errs.ErrSocialGiftPrivacyPassword
		}
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return "", err
	}
	token := hex.EncodeToString(secret)
	value.TokenHash = socialGiftTokenHash(token)
	value.ExpiresAt = time.Now().Add(12 * time.Hour).Unix()
	_, err = s.UserDataDB(uid).NewSession(c).ID(uid).Cols("token_hash", "expires_at").Update(value)
	return token, err
}

func socialGiftTokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (s *SocialGiftService) CheckPrivacy(c core.Context, uid int64, token string) error {
	if token == "" {
		return errs.ErrSocialGiftPrivacyLocked
	}
	value, err := s.GetPrivacy(c, uid)
	if err != nil {
		return err
	}
	if value == nil || value.ExpiresAt <= time.Now().Unix() || value.TokenHash != socialGiftTokenHash(token) {
		return errs.ErrSocialGiftPrivacyLocked
	}
	return nil
}

func (s *SocialGiftService) LockPrivacy(c core.Context, uid int64, token string) error {
	if token == "" {
		return nil
	}
	_, err := s.UserDataDB(uid).NewSession(c).Where("uid=? AND token_hash=?", uid, socialGiftTokenHash(token)).Cols("token_hash", "expires_at").Update(&models.SocialGiftPrivacy{})
	return err
}
