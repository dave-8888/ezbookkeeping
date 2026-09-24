package openai

import (
	"encoding/json"
	"github.com/mayswind/financially_free/pkg/core"
	"github.com/mayswind/financially_free/pkg/llm/data"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNativeToolCallRoundTrip(t *testing.T) {
	p := &CommonOpenAIChatCompletionsAPILargeLanguageModelAdapter{apiProvider: &OpenAIOfficialChatCompletionsAPIProvider{OpenAIModelID: "test"}}
	r, err := p.ParseTextualResponse(core.NewNullContext(), 1, []byte(`{"choices":[{"message":{"content":null,"reasoning_content":"opaque provider context","tool_calls":[{"id":"x","type":"function","function":{"name":"list_accounts","arguments":"{\"include_hidden\":true}"}}]}}]}`), data.LARGE_LANGUAGE_MODEL_RESPONSE_FORMAT_TOOLS)
	require.NoError(t, err)
	require.Len(t, r.Message.ToolCalls, 1)
	require.NotNil(t, r.Message.ReasoningContent)
	require.Equal(t, "opaque provider context", *r.Message.ReasoningContent)
	request := &data.LargeLanguageModelRequest{SystemPrompt: "test", Tools: []data.ToolDefinition{{Name: "list_accounts", Parameters: json.RawMessage(`{"type":"object"}`)}}, Messages: []data.ChatMessage{{Role: "user", Content: "看看资产"}, *r.Message, {Role: "tool", ToolCallID: "x", ToolName: "list_accounts", Content: `{"balance":"6000.00"}`}}}
	body, err := p.buildJsonRequestBody(core.NewNullContext(), 1, request, data.LARGE_LANGUAGE_MODEL_RESPONSE_FORMAT_TOOLS)
	require.NoError(t, err)
	var parsed map[string]any
	require.NoError(t, json.Unmarshal(body, &parsed))
	require.NotContains(t, parsed, "response_format")
	require.Len(t, parsed["tools"], 1)
	messages := parsed["messages"].([]any)
	require.Equal(t, "opaque provider context", messages[2].(map[string]any)["reasoning_content"])
	require.Equal(t, "x", messages[3].(map[string]any)["tool_call_id"])
	require.Equal(t, `{"include_hidden":true}`, messages[2].(map[string]any)["tool_calls"].([]any)[0].(map[string]any)["function"].(map[string]any)["arguments"])
}
