package api

import (
	"github.com/mayswind/financially_free/pkg/models"
	"testing"
)

func TestCompatibleKeyIsBoundToSavedEndpoint(t *testing.T) {
	api := &UserLLMSettingsApi{}
	saved := &models.UserLLMSetting{OpenAICompatibleBaseURL: "https://first.example/v1", OpenAICompatibleAPIKey: "saved-key"}
	for _, tc := range []struct{ url, key, want string }{
		{"https://first.example/v1/", "", "saved-key"},
		{"https://other.example/v1", "", ""},
		{"https://first.example/other", "", ""},
		{"https://other.example/v1", "new-key", "new-key"},
	} {
		req := &models.UserLLMSettingUpdateRequest{OpenAICompatibleBaseURL: tc.url, OpenAICompatibleAPIKey: tc.key}
		got := api.createUserLLMSettingModel(1, req, saved)
		if got.OpenAICompatibleAPIKey != tc.want {
			t.Error("incorrect saved key reuse")
		}
	}
}
