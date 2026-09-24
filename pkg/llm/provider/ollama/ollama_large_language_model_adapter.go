package ollama

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mayswind/financially_free/pkg/core"
	"github.com/mayswind/financially_free/pkg/errs"
	"github.com/mayswind/financially_free/pkg/llm/data"
	"github.com/mayswind/financially_free/pkg/llm/provider"
	"github.com/mayswind/financially_free/pkg/llm/provider/common"
	"github.com/mayswind/financially_free/pkg/log"
	"github.com/mayswind/financially_free/pkg/settings"
)

const ollamaChatCompletionsPath = "api/chat"

// OllamaLargeLanguageModelAdapter defines the structure of Ollama large language model adapter
type OllamaLargeLanguageModelAdapter struct {
	common.HttpLargeLanguageModelAdapter
	OllamaServerURL string
	OllamaModelID   string
}

// OllamaMessageRole defines the role of Ollama chat message
type OllamaMessageRole string

const (
	OllamaMessageRoleSystem OllamaMessageRole = "system"
	OllamaMessageRoleUser   OllamaMessageRole = "user"
)

// OllamaChatRequest defines the structure of Ollama chat request
type OllamaChatRequest struct {
	Tools    []any                       `json:"tools,omitempty"`
	Model    string                      `json:"model"`
	Stream   bool                        `json:"stream"`
	Messages []*OllamaChatRequestMessage `json:"messages"`
	Format   string                      `json:"format,omitempty"`
}

// OllamaChatRequestMessage defines the structure of Ollama chat request message
type OllamaChatRequestMessage struct {
	ToolCalls []any             `json:"tool_calls,omitempty"`
	ToolName  string            `json:"tool_name,omitempty"`
	Role      OllamaMessageRole `json:"role"`
	Content   string            `json:"content"`
	Images    []string          `json:"images,omitempty"`
}

// OllamaChatResponse defines the structure of Ollama chat response
type OllamaChatResponse struct {
	Message *OllamaChatResponseMessage `json:"message"`
}

// OllamaChatResponseMessage defines the structure of Ollama chat response message
type OllamaChatResponseMessage struct {
	Content   *string `json:"content"`
	ToolCalls []struct {
		Function struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		} `json:"function"`
	} `json:"tool_calls"`
}

// BuildTextualRequest returns the http request by Ollama large language model adapter
func (p *OllamaLargeLanguageModelAdapter) BuildTextualRequest(c core.Context, uid int64, request *data.LargeLanguageModelRequest, responseType data.LargeLanguageModelResponseFormat) (*http.Request, error) {
	requestBody, err := p.buildJsonRequestBody(c, uid, request, responseType)

	if err != nil {
		return nil, err
	}

	httpRequest, err := http.NewRequest("POST", p.getOllamaRequestUrl(), bytes.NewReader(requestBody))

	if err != nil {
		return nil, err
	}

	httpRequest.Header.Set("Content-Type", "application/json")

	return httpRequest, nil
}

// ParseTextualResponse returns the textual response by Ollama large language model adapter
func (p *OllamaLargeLanguageModelAdapter) ParseTextualResponse(c core.Context, uid int64, body []byte, responseType data.LargeLanguageModelResponseFormat) (*data.LargeLanguageModelTextualResponse, error) {
	chatResponse := &OllamaChatResponse{}
	err := json.Unmarshal(body, &chatResponse)

	if err != nil {
		log.Errorf(c, "[ollama_large_language_model_adapter.ParseTextualResponse] failed to parse chat response for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.ErrFailedToRequestRemoteApi
	}

	if chatResponse == nil || chatResponse.Message == nil || (chatResponse.Message.Content == nil && len(chatResponse.Message.ToolCalls) == 0) {
		log.Errorf(c, "[ollama_large_language_model_adapter.ParseTextualResponse] chat response is invalid for user \"uid:%d\"", uid)
		return nil, errs.ErrFailedToRequestRemoteApi
	}

	message := &data.ChatMessage{Role: "assistant"}
	if chatResponse.Message.Content != nil {
		message.Content = *chatResponse.Message.Content
	}
	for i, call := range chatResponse.Message.ToolCalls {
		message.ToolCalls = append(message.ToolCalls, data.ToolCall{ID: fmt.Sprintf("call_%d", i), Name: call.Function.Name, Arguments: call.Function.Arguments})
	}
	textualResponse := &data.LargeLanguageModelTextualResponse{Content: message.Content}
	if responseType == data.LARGE_LANGUAGE_MODEL_RESPONSE_FORMAT_TOOLS {
		textualResponse.Message = message
	}

	return textualResponse, nil
}

func (p *OllamaLargeLanguageModelAdapter) buildJsonRequestBody(c core.Context, uid int64, request *data.LargeLanguageModelRequest, responseType data.LargeLanguageModelResponseFormat) ([]byte, error) {
	if p.OllamaModelID == "" {
		return nil, errs.ErrInvalidLLMModelId
	}

	chatRequest := &OllamaChatRequest{
		Model:    p.OllamaModelID,
		Stream:   request.Stream,
		Messages: make([]*OllamaChatRequestMessage, 0, 2),
	}

	if request.SystemPrompt != "" {
		chatRequest.Messages = append(chatRequest.Messages, &OllamaChatRequestMessage{
			Role:    OllamaMessageRoleSystem,
			Content: request.SystemPrompt,
		})
	}

	if len(request.UserPrompt) > 0 {
		if request.UserPromptType == data.LARGE_LANGUAGE_MODEL_REQUEST_PROMPT_TYPE_IMAGE_URL {
			imageBase64Data := base64.StdEncoding.EncodeToString(request.UserPrompt)
			chatRequest.Messages = append(chatRequest.Messages, &OllamaChatRequestMessage{
				Role:   OllamaMessageRoleUser,
				Images: []string{imageBase64Data},
			})
		} else {
			chatRequest.Messages = append(chatRequest.Messages, &OllamaChatRequestMessage{
				Role:    OllamaMessageRoleUser,
				Content: string(request.UserPrompt),
			})
		}
	}

	if responseType == data.LARGE_LANGUAGE_MODEL_RESPONSE_FORMAT_TOOLS {
		for _, message := range request.Messages {
			item := &OllamaChatRequestMessage{Role: OllamaMessageRole(message.Role), Content: message.Content, ToolName: message.ToolName}
			for _, call := range message.ToolCalls {
				item.ToolCalls = append(item.ToolCalls, map[string]any{"function": map[string]any{"name": call.Name, "arguments": call.Arguments}})
			}
			chatRequest.Messages = append(chatRequest.Messages, item)
		}
		for _, tool := range request.Tools {
			chatRequest.Tools = append(chatRequest.Tools, map[string]any{"type": "function", "function": tool})
		}
	}
	if responseType == data.LARGE_LANGUAGE_MODEL_RESPONSE_FORMAT_JSON {
		chatRequest.Format = "json"
	}

	requestBodyBytes, err := json.Marshal(chatRequest)

	if err != nil {
		log.Errorf(c, "[ollama_large_language_model_adapter.buildJsonRequestBody] failed to marshal request body for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.ErrOperationFailed
	}

	if responseType != data.LARGE_LANGUAGE_MODEL_RESPONSE_FORMAT_TOOLS {
		log.Debugf(c, "[ollama_large_language_model_adapter.buildJsonRequestBody] request body is %s", requestBodyBytes)
	}
	return requestBodyBytes, nil
}

func (p *OllamaLargeLanguageModelAdapter) getOllamaRequestUrl() string {
	url := p.OllamaServerURL

	if url[len(url)-1] != '/' {
		url += "/"
	}

	url += ollamaChatCompletionsPath
	return url
}

// NewOllamaLargeLanguageModelProvider creates a new Ollama large language model provider instance
func NewOllamaLargeLanguageModelProvider(llmConfig *settings.LLMConfig) provider.LargeLanguageModelProvider {
	return common.NewCommonHttpLargeLanguageModelProvider(llmConfig, &OllamaLargeLanguageModelAdapter{
		OllamaServerURL: llmConfig.OllamaServerURL,
		OllamaModelID:   llmConfig.OllamaModelID,
	})
}
