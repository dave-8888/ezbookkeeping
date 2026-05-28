package avatars

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mayswind/financially_free/pkg/models"
)

func TestNullAvatarProvider_GetGravatarUrl(t *testing.T) {
	avatarProvider := NewNullAvatarProvider()

	expectedValue := ""
	actualValue := avatarProvider.GetAvatarUrl(&models.User{
		Email: "MyEmailAddress@example.com",
	})

	assert.Equal(t, expectedValue, actualValue)
}
