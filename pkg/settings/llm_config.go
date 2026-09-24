package settings

import "strings"

// IsConfigured reports whether the provider has the required connection settings.
// It does not probe the provider or validate credentials remotely.
func (c *LLMConfig) IsConfigured() bool {
	if c == nil {
		return false
	}
	present := func(values ...string) bool {
		for _, value := range values {
			if strings.TrimSpace(value) == "" {
				return false
			}
		}
		return true
	}
	switch c.LLMProvider {
	case OpenAILLMProvider:
		return present(c.OpenAIAPIKey, c.OpenAIModelID)
	case OpenAICompatibleLLMProvider:
		return present(c.OpenAICompatibleBaseURL, c.OpenAICompatibleModelID)
	case OpenRouterLLMProvider:
		return present(c.OpenRouterAPIKey, c.OpenRouterModelID)
	case OllamaLLMProvider:
		return present(c.OllamaServerURL, c.OllamaModelID)
	case GoogleAILLMProvider:
		return present(c.GoogleAIAPIKey, c.GoogleAIModelID)
	default:
		return false
	}
}
