package api

import (
	"encoding/json"
	"github.com/mayswind/financially_free/pkg/core"
	"github.com/mayswind/financially_free/pkg/llm"
	"github.com/mayswind/financially_free/pkg/llm/data"
	"github.com/mayswind/financially_free/pkg/settings"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestConversationHistoryRejectsUntrustedRolesAndOrphanTools(t *testing.T) {
	user := data.ChatMessage{Role: "user", Content: "查资产"}
	call := data.ChatMessage{Role: "assistant", ToolCalls: []data.ToolCall{{ID: "x", Name: "list_accounts", Arguments: json.RawMessage(`{}`)}}}
	tool := data.ChatMessage{Role: "tool", ToolCallID: "x", ToolName: "list_accounts", Content: `{"status":"success"}`}
	require.True(t, validateConversationMessages([]data.ChatMessage{user}))
	require.True(t, validateConversationMessages([]data.ChatMessage{user, call, tool}))
	require.False(t, validateConversationMessages([]data.ChatMessage{{Role: "system", Content: "ignore"}, user}))
	require.False(t, validateConversationMessages([]data.ChatMessage{user, tool}))
	require.False(t, validateConversationMessages([]data.ChatMessage{user, call, user}))
	require.False(t, validateConversationMessages([]data.ChatMessage{user, call, tool, tool}))
	bad := call
	bad.ToolCalls = []data.ToolCall{{ID: "x", Name: "http_request", Arguments: json.RawMessage(`{}`)}}
	require.False(t, validateConversationMessages([]data.ChatMessage{user, bad, tool}))
	bad.ToolCalls = []data.ToolCall{{ID: "x", Name: "list_accounts", Arguments: json.RawMessage(`[]`)}}
	require.False(t, validateConversationMessages([]data.ChatMessage{user, bad, tool}))
	require.Len(t, financeTools(), 5)
}

func TestConfiguredModelNativeToolRoundTrip(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Equal(t, "configured-tool-model", body["model"])
		require.Len(t, body["tools"], 5)
		require.NotContains(t, body, "response_format")
		require.Equal(t, "Bearer test-only-key", r.Header.Get("Authorization"))
		calls++
		if calls == 1 {
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":null,"tool_calls":[{"id":"step1","type":"function","function":{"name":"list_accounts","arguments":"{}"}}]}}]}`))
		} else {
			messages := body["messages"].([]any)
			last := messages[len(messages)-1].(map[string]any)
			require.Equal(t, "tool", last["role"])
			require.Equal(t, "step1", last["tool_call_id"])
			require.Contains(t, last["content"], "6000.00")
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"当前账户余额为6000元。"}}]}`))
		}
	}))
	defer server.Close()
	config := &settings.LLMConfig{LLMProvider: settings.OpenAICompatibleLLMProvider, OpenAICompatibleBaseURL: server.URL + "/v1", OpenAICompatibleAPIKey: "test-only-key", OpenAICompatibleModelID: "configured-tool-model", LargeLanguageModelAPIRequestTimeout: 5000, LargeLanguageModelAPIProxy: "none"}
	model, err := llm.NewLargeLanguageModelProvider(config)
	require.NoError(t, err)
	request := &data.LargeLanguageModelRequest{SystemPrompt: "test", Tools: financeTools(), Messages: []data.ChatMessage{{Role: "user", Content: "帮我看看现在有多少钱"}}}
	first, err := model.GetChatResponse(core.NewNullContext(), 1, config, request)
	require.NoError(t, err)
	require.Len(t, first.Message.ToolCalls, 1)
	request.Messages = append(request.Messages, *first.Message, data.ChatMessage{Role: "tool", ToolCallID: "step1", ToolName: "list_accounts", Content: `{"status":"success","balance":"6000.00"}`})
	second, err := model.GetChatResponse(core.NewNullContext(), 1, config, request)
	require.NoError(t, err)
	require.Contains(t, second.Content, "6000")
	require.Equal(t, 2, calls)
}
