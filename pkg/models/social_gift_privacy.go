package models

// SocialGiftPrivacy stores only password and access-token hashes, never plaintext secrets.
type SocialGiftPrivacy struct {
	Uid          int64  `xorm:"PK"`
	PasswordHash string `xorm:"VARCHAR(128) NOT NULL"`
	TokenHash    string `xorm:"VARCHAR(64)"`
	ExpiresAt    int64
	RetryAfter   int64
}
