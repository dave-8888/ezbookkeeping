package googleai

import (
	"encoding/json"
	"github.com/mayswind/financially_free/pkg/llm/data"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNativeToolSignaturesAndResultsSurviveRoundTrip(t *testing.T) {
	r, err := parseToolResponse([]byte(`{"candidates":[{"content":{"parts":[{"text":"private reasoning","thought":true,"thoughtSignature":"text-signature"},{"functionCall":{"name":"list_accounts","args":{}},"thoughtSignature":"call-signature"}]}}]}`))
	require.NoError(t, err)
	require.Empty(t, r.Content)
	require.Len(t, r.Message.ToolCalls, 1)
	p := &GoogleAILargeLanguageModelAdapter{GoogleAIModelID: "test"}
	body, err := p.buildToolRequest(&data.LargeLanguageModelRequest{SystemPrompt: "trusted", Tools: []data.ToolDefinition{{Name: "list_accounts", Parameters: json.RawMessage(`{"type":"object"}`)}}, Messages: []data.ChatMessage{{Role: "user", Content: "资产"}, *r.Message, {Role: "tool", ToolName: "list_accounts", Content: `{"status":"success"}`}}})
	require.NoError(t, err)
	var parsed map[string]any
	require.NoError(t, json.Unmarshal(body, &parsed))
	contents := parsed["contents"].([]any)
	parts := contents[1].(map[string]any)["parts"].([]any)
	require.Equal(t, "text-signature", parts[0].(map[string]any)["thoughtSignature"])
	require.Equal(t, "call-signature", parts[1].(map[string]any)["thoughtSignature"])
	require.Equal(t, "model", contents[1].(map[string]any)["role"])
	require.Equal(t, "list_accounts", contents[2].(map[string]any)["parts"].([]any)[0].(map[string]any)["functionResponse"].(map[string]any)["name"])
}
