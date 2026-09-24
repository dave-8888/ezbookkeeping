package ollama

import (
	"encoding/json"
	"github.com/mayswind/financially_free/pkg/core"
	"github.com/mayswind/financially_free/pkg/llm/data"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNativeToolsUseObjectArgumentsAndNamedResults(t *testing.T) {
	p := &OllamaLargeLanguageModelAdapter{OllamaModelID: "test"}
	r, err := p.ParseTextualResponse(core.NewNullContext(), 1, []byte(`{"message":{"tool_calls":[{"function":{"name":"list_accounts","arguments":{"include_hidden":true}}}]}}`), data.LARGE_LANGUAGE_MODEL_RESPONSE_FORMAT_TOOLS)
	require.NoError(t, err)
	require.Len(t, r.Message.ToolCalls, 1)
	body, err := p.buildJsonRequestBody(core.NewNullContext(), 1, &data.LargeLanguageModelRequest{Tools: []data.ToolDefinition{{Name: "list_accounts", Parameters: json.RawMessage(`{"type":"object"}`)}}, Messages: []data.ChatMessage{*r.Message, {Role: "tool", ToolName: "list_accounts", ToolCallID: r.Message.ToolCalls[0].ID, Content: `{"status":"success"}`}}}, data.LARGE_LANGUAGE_MODEL_RESPONSE_FORMAT_TOOLS)
	require.NoError(t, err)
	var parsed map[string]any
	require.NoError(t, json.Unmarshal(body, &parsed))
	require.NotContains(t, parsed, "format")
	messages := parsed["messages"].([]any)
	require.Equal(t, "list_accounts", messages[1].(map[string]any)["tool_name"])
	require.Equal(t, true, messages[0].(map[string]any)["tool_calls"].([]any)[0].(map[string]any)["function"].(map[string]any)["arguments"].(map[string]any)["include_hidden"])
}
