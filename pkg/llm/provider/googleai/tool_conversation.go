package googleai

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mayswind/financially_free/pkg/errs"
	"github.com/mayswind/financially_free/pkg/llm/data"
)

func (p *GoogleAILargeLanguageModelAdapter) buildToolRequest(request *data.LargeLanguageModelRequest) ([]byte, error) {
	if p.GoogleAIModelID == "" {
		return nil, errs.ErrInvalidLLMModelId
	}
	contents := []any{}
	responseIDs := map[string]string{}
	for _, message := range request.Messages {
		role := "user"
		parts := []any{}
		if message.Role == "assistant" {
			role = "model"
			if len(message.NativeParts) > 0 {
				// Thought signatures must survive the round trip unchanged.
				var native []struct {
					Call *struct {
						ID string `json:"id"`
					} `json:"functionCall"`
				}
				if json.Unmarshal(message.NativeParts, &native) == nil {
					index := 0
					for _, part := range native {
						if part.Call != nil {
							if index < len(message.ToolCalls) && part.Call.ID != "" {
								responseIDs[message.ToolCalls[index].ID] = part.Call.ID
							}
							index++
						}
					}
				}
				contents = append(contents, map[string]any{"role": role, "parts": message.NativeParts})
				continue
			}
		}
		if message.Role == "tool" {
			var result any
			if json.Unmarshal([]byte(message.Content), &result) != nil {
				result = map[string]any{"text": message.Content}
			}
			response := map[string]any{"name": message.ToolName, "response": map[string]any{"result": result}}
			if id := responseIDs[message.ToolCallID]; id != "" {
				response["id"] = id
			}
			parts = append(parts, map[string]any{"functionResponse": response})
		} else {
			if message.Content != "" {
				parts = append(parts, map[string]any{"text": message.Content})
			}
			for _, call := range message.ToolCalls {
				parts = append(parts, map[string]any{"functionCall": map[string]any{"name": call.Name, "args": call.Arguments}})
			}
		}
		if len(parts) == 0 {
			parts = append(parts, map[string]any{"text": " "})
		}
		// Gemini expects all results of parallel calls in a single user content.
		if message.Role == "tool" && len(contents) > 0 {
			last := contents[len(contents)-1].(map[string]any)
			if last["role"] == "user" {
				if previous, ok := last["parts"].([]any); ok {
					last["parts"] = append(previous, parts...)
					continue
				}
			}
		}
		contents = append(contents, map[string]any{"role": role, "parts": parts})
	}
	return json.Marshal(map[string]any{
		"systemInstruction": map[string]any{"parts": []any{map[string]any{"text": request.SystemPrompt}}},
		"contents":          contents,
		"tools":             []any{map[string]any{"functionDeclarations": request.Tools}},
	})
}

func parseToolResponse(body []byte) (*data.LargeLanguageModelTextualResponse, error) {
	var response struct {
		Candidates []struct {
			Content struct {
				Parts json.RawMessage `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if json.Unmarshal(body, &response) != nil || len(response.Candidates) == 0 {
		return nil, errs.ErrFailedToRequestRemoteApi
	}
	raw := response.Candidates[0].Content.Parts
	var parts []struct {
		Text    string `json:"text"`
		Thought bool   `json:"thought"`
		Call    *struct {
			Name string          `json:"name"`
			Args json.RawMessage `json:"args"`
		} `json:"functionCall"`
	}
	if json.Unmarshal(raw, &parts) != nil {
		return nil, errs.ErrFailedToRequestRemoteApi
	}
	message := &data.ChatMessage{Role: "assistant", NativeParts: raw}
	var text strings.Builder
	for i, part := range parts {
		if !part.Thought {
			text.WriteString(part.Text)
		}
		if part.Call != nil {
			message.ToolCalls = append(message.ToolCalls, data.ToolCall{ID: fmt.Sprintf("call_%d", i), Name: part.Call.Name, Arguments: part.Call.Args})
		}
	}
	message.Content = text.String()
	if message.Content == "" && len(message.ToolCalls) == 0 {
		return nil, errs.ErrFailedToRequestRemoteApi
	}
	return &data.LargeLanguageModelTextualResponse{Content: message.Content, Message: message}, nil
}
