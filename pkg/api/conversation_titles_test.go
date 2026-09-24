package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/mayswind/financially_free/pkg/core"
	"github.com/mayswind/financially_free/pkg/settings"
	"github.com/stretchr/testify/require"
)

func TestConversationTitleOutputValidation(t *testing.T) {
	for _, content := range []string{`null`, `{}`, `{"title":""}`, `{"title":"午餐支出"}`, `{"title":"9月14日午餐"}`, `{"title":"9月13日"}`, `not json`} {
		_, ok := normalizeConversationTitle(content, "9月13日")
		require.False(t, ok, content)
	}
	title, ok := normalizeConversationTitle(`{"title":" 9月13日 午餐支出与交通购物娱乐日常开销🚗 "}`, "9月13日")
	require.True(t, ok)
	require.Equal(t, 15, utf8.RuneCountInString(title))
	require.True(t, utf8.ValidString(title))
	title, ok = normalizeConversationTitle(`{"title":"9月13日🚗出行"}`, "9月13日")
	require.True(t, ok)
	require.Equal(t, "9月13日🚗出行", title)
}

func TestConversationTitleUsesConfiguredModelAndDate(t *testing.T) {
	var received struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer title-test-key" {
			t.Errorf("unexpected model request: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Error(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"title\":\"9月13日午餐支出\"}"}}]}`))
	}))
	defer server.Close()
	config := &settings.LLMConfig{LLMProvider: settings.OpenAICompatibleLLMProvider,
		OpenAICompatibleBaseURL: server.URL + "/v1", OpenAICompatibleAPIKey: "title-test-key",
		OpenAICompatibleModelID: "configured-model", LargeLanguageModelAPIRequestTimeout: 5000, LargeLanguageModelAPIProxy: "none"}
	result, err := generateConversationTitle(core.NewNullContext(), 42, config, time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC), "午饭30.5元")
	require.Nil(t, err)
	require.Equal(t, "9月13日午餐支出", result.Title)
	require.Equal(t, "configured-model", received.Model)
	require.Len(t, received.Messages, 2)
	require.Contains(t, received.Messages[0].Content, "15")
	var prompt conversationTitleRequest
	require.NoError(t, json.Unmarshal([]byte(received.Messages[1].Content), &prompt))
	require.Equal(t, "2026-09-13", prompt.Date)
	require.Equal(t, "午饭30.5元", prompt.Text)
}

func TestConversationTitleRejectsBadInputBeforeAccessingServices(t *testing.T) {
	a := &LargeLanguageModelsApi{}
	for _, body := range []string{`{}`, `{"date":"2026-02-30","text":"午餐"}`, `{"date":"2026-09-13","text":" "}`, `{"date":"2026-09-13","text":"` + strings.Repeat("长", 2001) + `"}`} {
		_, err := a.GenerateConversationTitleHandler(newAssetPlanTestWebContext(http.MethodPost, body, 42))
		require.NotNil(t, err)
	}
}
