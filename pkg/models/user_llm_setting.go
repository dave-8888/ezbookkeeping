package models

import "github.com/mayswind/financially_free/pkg/settings"

// UserLLMSetting represents user large language model settings stored in database
type UserLLMSetting struct {
	Uid                                 int64 `xorm:"PK"`
	Enabled                             bool
	LLMProvider                         string
	OpenAIAPIKey                        string
	OpenAIModelID                       string
	OpenAICompatibleBaseURL             string
	OpenAICompatibleAPIKey              string
	OpenAICompatibleModelID             string
	OpenRouterAPIKey                    string
	OpenRouterModelID                   string
	OllamaServerURL                     string
	OllamaModelID                       string
	GoogleAIAPIKey                      string
	GoogleAIModelID                     string
	LargeLanguageModelAPIRequestTimeout uint32
	LargeLanguageModelAPIProxy          string
	LargeLanguageModelAPISkipTLSVerify  bool
	UpdatedUnixTime                     int64
}

// UserLLMSettingResponse represents user large language model settings returned to frontend
type UserLLMSettingResponse struct {
	Enabled                             bool   `json:"enabled"`
	LLMProvider                         string `json:"llmProvider"`
	OpenAIAPIKeySet                     bool   `json:"openAIAPIKeySet"`
	OpenAIModelID                       string `json:"openAIModelID"`
	OpenAICompatibleBaseURL             string `json:"openAICompatibleBaseURL"`
	OpenAICompatibleAPIKeySet           bool   `json:"openAICompatibleAPIKeySet"`
	OpenAICompatibleModelID             string `json:"openAICompatibleModelID"`
	OpenRouterAPIKeySet                 bool   `json:"openRouterAPIKeySet"`
	OpenRouterModelID                   string `json:"openRouterModelID"`
	OllamaServerURL                     string `json:"ollamaServerURL"`
	OllamaModelID                       string `json:"ollamaModelID"`
	GoogleAIAPIKeySet                   bool   `json:"googleAIAPIKeySet"`
	GoogleAIModelID                     string `json:"googleAIModelID"`
	LargeLanguageModelAPIRequestTimeout uint32 `json:"largeLanguageModelAPIRequestTimeout"`
	LargeLanguageModelAPIProxy          string `json:"largeLanguageModelAPIProxy"`
	LargeLanguageModelAPISkipTLSVerify  bool   `json:"largeLanguageModelAPISkipTLSVerify"`
	UpdatedUnixTime                     int64  `json:"updatedUnixTime"`
}

// UserLLMSettingUpdateRequest represents all parameters of user large language model settings update request
type UserLLMSettingUpdateRequest struct {
	Enabled                             bool   `json:"enabled"`
	LLMProvider                         string `json:"llmProvider"`
	OpenAIAPIKey                        string `json:"openAIAPIKey"`
	OpenAIModelID                       string `json:"openAIModelID"`
	OpenAICompatibleBaseURL             string `json:"openAICompatibleBaseURL"`
	OpenAICompatibleAPIKey              string `json:"openAICompatibleAPIKey"`
	OpenAICompatibleModelID             string `json:"openAICompatibleModelID"`
	OpenRouterAPIKey                    string `json:"openRouterAPIKey"`
	OpenRouterModelID                   string `json:"openRouterModelID"`
	OllamaServerURL                     string `json:"ollamaServerURL"`
	OllamaModelID                       string `json:"ollamaModelID"`
	GoogleAIAPIKey                      string `json:"googleAIAPIKey"`
	GoogleAIModelID                     string `json:"googleAIModelID"`
	LargeLanguageModelAPIRequestTimeout uint32 `json:"largeLanguageModelAPIRequestTimeout"`
	LargeLanguageModelAPIProxy          string `json:"largeLanguageModelAPIProxy"`
	LargeLanguageModelAPISkipTLSVerify  bool   `json:"largeLanguageModelAPISkipTLSVerify"`
}

// ToUserLLMSettingResponse returns a view-object of the user llm setting
func (s *UserLLMSetting) ToUserLLMSettingResponse() *UserLLMSettingResponse {
	if s == nil {
		return &UserLLMSettingResponse{
			LargeLanguageModelAPIProxy: "system",
		}
	}

	return &UserLLMSettingResponse{
		Enabled:                             s.Enabled,
		LLMProvider:                         s.LLMProvider,
		OpenAIAPIKeySet:                     s.OpenAIAPIKey != "",
		OpenAIModelID:                       s.OpenAIModelID,
		OpenAICompatibleBaseURL:             s.OpenAICompatibleBaseURL,
		OpenAICompatibleAPIKeySet:           s.OpenAICompatibleAPIKey != "",
		OpenAICompatibleModelID:             s.OpenAICompatibleModelID,
		OpenRouterAPIKeySet:                 s.OpenRouterAPIKey != "",
		OpenRouterModelID:                   s.OpenRouterModelID,
		OllamaServerURL:                     s.OllamaServerURL,
		OllamaModelID:                       s.OllamaModelID,
		GoogleAIAPIKeySet:                   s.GoogleAIAPIKey != "",
		GoogleAIModelID:                     s.GoogleAIModelID,
		LargeLanguageModelAPIRequestTimeout: s.LargeLanguageModelAPIRequestTimeout,
		LargeLanguageModelAPIProxy:          s.LargeLanguageModelAPIProxy,
		LargeLanguageModelAPISkipTLSVerify:  s.LargeLanguageModelAPISkipTLSVerify,
		UpdatedUnixTime:                     s.UpdatedUnixTime,
	}
}

// ToLLMConfig converts user llm setting to runtime llm config
func (s *UserLLMSetting) ToLLMConfig(defaultConfig *settings.LLMConfig) *settings.LLMConfig {
	if s == nil || !s.Enabled || s.LLMProvider == "" {
		return nil
	}

	requestTimeout := s.LargeLanguageModelAPIRequestTimeout
	apiProxy := s.LargeLanguageModelAPIProxy

	if defaultConfig != nil {
		if requestTimeout == 0 {
			requestTimeout = defaultConfig.LargeLanguageModelAPIRequestTimeout
		}

		if apiProxy == "" {
			apiProxy = defaultConfig.LargeLanguageModelAPIProxy
		}
	}

	if apiProxy == "" {
		apiProxy = "system"
	}

	return &settings.LLMConfig{
		LLMProvider:                         s.LLMProvider,
		OpenAIAPIKey:                        s.OpenAIAPIKey,
		OpenAIModelID:                       s.OpenAIModelID,
		OpenAICompatibleBaseURL:             s.OpenAICompatibleBaseURL,
		OpenAICompatibleAPIKey:              s.OpenAICompatibleAPIKey,
		OpenAICompatibleModelID:             s.OpenAICompatibleModelID,
		OpenRouterAPIKey:                    s.OpenRouterAPIKey,
		OpenRouterModelID:                   s.OpenRouterModelID,
		OllamaServerURL:                     s.OllamaServerURL,
		OllamaModelID:                       s.OllamaModelID,
		GoogleAIAPIKey:                      s.GoogleAIAPIKey,
		GoogleAIModelID:                     s.GoogleAIModelID,
		LargeLanguageModelAPIRequestTimeout: requestTimeout,
		LargeLanguageModelAPIProxy:          apiProxy,
		LargeLanguageModelAPISkipTLSVerify:  s.LargeLanguageModelAPISkipTLSVerify,
	}
}
