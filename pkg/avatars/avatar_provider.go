package avatars

import "github.com/mayswind/financially_free/pkg/models"

// AvatarProvider is user avatar provider interface
type AvatarProvider interface {
	GetAvatarUrl(user *models.User) string
}
