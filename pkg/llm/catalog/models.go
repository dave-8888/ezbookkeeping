package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/mayswind/financially_free/pkg/settings"
	"github.com/mayswind/financially_free/pkg/utils"
)

// ListModels fetches available model IDs, without performing inference or saving settings.
func ListModels(ctx context.Context, config *settings.LLMConfig) ([]string, error) {
	endpoint, key, err := modelEndpoint(config)
	if err != nil {
		return nil, err
	}
	timeout := config.LargeLanguageModelAPIRequestTimeout
	if timeout == 0 || timeout > 60000 {
		timeout = 60000
	}
	client := utils.NewHttpClient(timeout, config.LargeLanguageModelAPIProxy, config.LargeLanguageModelAPISkipTLSVerify, settings.GetUserAgent())
	defer client.CloseIdleConnections()
	// Never forward saved credentials through redirects, including same-domain ones.
	client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
	ids := make(map[string]bool)
	pageToken := ""
	for page := 0; page < 20; page++ {
		u, _ := url.Parse(endpoint)
		if pageToken != "" {
			q := u.Query()
			q.Set("pageToken", pageToken)
			u.RawQuery = q.Encode()
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, fmt.Errorf("invalid model endpoint")
		}
		if key != "" {
			if config.LLMProvider == settings.GoogleAILLMProvider {
				req.Header.Set("x-goog-api-key", key)
			} else {
				req.Header.Set("Authorization", "Bearer "+key)
			}
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("model list request failed")
		}
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024+1))
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("model list returned HTTP %d", resp.StatusCode)
		}
		if readErr != nil || len(body) > 2*1024*1024 {
			return nil, fmt.Errorf("invalid model list response")
		}
		models, next, err := parseModels(body, config.LLMProvider)
		if err != nil {
			return nil, err
		}
		for _, id := range models {
			ids[id] = true
		}
		if next == "" {
			result := make([]string, 0, len(ids))
			for id := range ids {
				result = append(result, id)
			}
			sort.Strings(result)
			return result, nil
		}
		if next == pageToken {
			return nil, fmt.Errorf("invalid model pagination")
		}
		pageToken = next
	}
	return nil, fmt.Errorf("too many model pages")
}

func modelEndpoint(c *settings.LLMConfig) (string, string, error) {
	if c == nil {
		return "", "", fmt.Errorf("missing AI settings")
	}
	base, key, path := "", "", "/models"
	switch c.LLMProvider {
	case settings.OpenAILLMProvider:
		base, key = "https://api.openai.com/v1", c.OpenAIAPIKey
	case settings.OpenAICompatibleLLMProvider:
		base, key = c.OpenAICompatibleBaseURL, c.OpenAICompatibleAPIKey
	case settings.OpenRouterLLMProvider:
		base, key = "https://openrouter.ai/api/v1", c.OpenRouterAPIKey
	case settings.OllamaLLMProvider:
		base, path = c.OllamaServerURL, "/api/tags"
	case settings.GoogleAILLMProvider:
		base, key = "https://generativelanguage.googleapis.com/v1beta", c.GoogleAIAPIKey
	default:
		return "", "", fmt.Errorf("unsupported provider")
	}
	if !ValidBaseURL(base) {
		return "", "", fmt.Errorf("invalid base URL")
	}
	if key == "" && c.LLMProvider != settings.OllamaLLMProvider && c.LLMProvider != settings.OpenAICompatibleLLMProvider {
		return "", "", fmt.Errorf("missing API key")
	}
	return strings.TrimRight(base, "/") + path, key, nil
}

func parseModels(body []byte, protocol string) ([]string, string, error) {
	var response struct {
		Data []struct {
			ID   string `json:"id"`
			Type string `json:"type"`
		} `json:"data"`
		Models []struct {
			Name                       string   `json:"name"`
			SupportedGenerationMethods []string `json:"supportedGenerationMethods"`
		} `json:"models"`
		NextPageToken string `json:"nextPageToken"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, "", fmt.Errorf("invalid model list JSON")
	}
	result := make([]string, 0)
	if protocol == settings.OllamaLLMProvider || protocol == settings.GoogleAILLMProvider {
		if response.Models == nil {
			return nil, "", fmt.Errorf("missing model list")
		}
		for _, model := range response.Models {
			if protocol == settings.GoogleAILLMProvider {
				supported := false
				for _, method := range model.SupportedGenerationMethods {
					if method == "generateContent" {
						supported = true
					}
				}
				if !supported {
					continue
				}
			}
			id := strings.TrimSpace(model.Name)
			if protocol == settings.GoogleAILLMProvider {
				id = strings.TrimPrefix(id, "models/")
			}
			if id != "" {
				result = append(result, id)
			}
		}
	} else {
		if response.Data == nil {
			return nil, "", fmt.Errorf("missing model list")
		}
		for _, model := range response.Data {
			// Some providers explicitly distinguish non-chat models.
			if model.Type != "" && model.Type != "chat" {
				continue
			}
			if id := strings.TrimSpace(model.ID); id != "" {
				result = append(result, id)
			}
		}
	}
	if protocol != settings.GoogleAILLMProvider {
		response.NextPageToken = ""
	}
	return result, response.NextPageToken, nil
}
