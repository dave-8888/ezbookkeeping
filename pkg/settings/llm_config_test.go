package settings

import "testing"

func TestLLMConfigIsConfigured(t *testing.T) {
	cases := []struct {
		name   string
		config *LLMConfig
		want   bool
	}{
		{"nil", nil, false},
		{"empty", &LLMConfig{}, false},
		{"provider only", &LLMConfig{LLMProvider: OpenAILLMProvider}, false},
		{"missing key", &LLMConfig{LLMProvider: OpenAILLMProvider, OpenAIModelID: "model"}, false},
		{"blank key", &LLMConfig{LLMProvider: OpenAILLMProvider, OpenAIModelID: "model", OpenAIAPIKey: " "}, false},
		{"openai", &LLMConfig{LLMProvider: OpenAILLMProvider, OpenAIModelID: "model", OpenAIAPIKey: "key"}, true},
		{"compatible without optional key", &LLMConfig{LLMProvider: OpenAICompatibleLLMProvider, OpenAICompatibleBaseURL: "http://localhost/v1", OpenAICompatibleModelID: "model"}, true},
		{"ollama", &LLMConfig{LLMProvider: OllamaLLMProvider, OllamaServerURL: "http://localhost:11434", OllamaModelID: "model"}, true},
		{"openrouter", &LLMConfig{LLMProvider: OpenRouterLLMProvider, OpenRouterAPIKey: "key", OpenRouterModelID: "model"}, true},
		{"google", &LLMConfig{LLMProvider: GoogleAILLMProvider, GoogleAIAPIKey: "key", GoogleAIModelID: "model"}, true},
		{"unknown", &LLMConfig{LLMProvider: "unknown"}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.config.IsConfigured(); got != c.want {
				t.Fatalf("IsConfigured() = %v, want %v", got, c.want)
			}
		})
	}
}
