package catalog

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
)

//go:embed providers.json
var defaults []byte

// Provider is a UI preset; protocol names remain compatible with saved settings.
type Provider struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	BaseURL  string `json:"baseUrl"`
	DocsURL  string `json:"docsUrl"`
}

// Load reads the catalog on every request so administrator edits need no restart.
// An external file replaces the full built-in catalog.
func Load(path string) ([]Provider, error) {
	body := defaults
	if path != "" {
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		if info.Size() > 1024*1024 {
			return nil, fmt.Errorf("provider catalog is too large")
		}
		body, err = os.ReadFile(path)
		if err != nil {
			return nil, err
		}
	}
	var providers []Provider
	if err := json.Unmarshal(body, &providers); err != nil {
		return nil, err
	}
	if len(providers) == 0 || len(providers) > 100 {
		return nil, fmt.Errorf("invalid provider count")
	}
	seen := make(map[string]bool)
	for _, p := range providers {
		if strings.TrimSpace(p.ID) == "" || strings.TrimSpace(p.Name) == "" || seen[p.ID] {
			return nil, fmt.Errorf("invalid provider id or name")
		}
		seen[p.ID] = true
		switch p.Protocol {
		case "openai", "openai_compatible", "openrouter", "ollama", "google_ai":
		default:
			return nil, fmt.Errorf("unsupported provider protocol")
		}
		if p.BaseURL != "" && !ValidBaseURL(p.BaseURL) {
			return nil, fmt.Errorf("invalid provider base URL")
		}
		if p.DocsURL != "" {
			u, err := url.Parse(p.DocsURL)
			if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil {
				return nil, fmt.Errorf("invalid documentation URL")
			}
		}
	}
	return providers, nil
}

// ValidBaseURL permits local model servers as well as hosted HTTP APIs.
func ValidBaseURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Hostname() != "" && u.User == nil && u.RawQuery == "" && u.Fragment == ""
}

// SameBaseURL prevents credentials from being reused for a different endpoint.
func SameBaseURL(a, b string) bool {
	return strings.TrimRight(strings.TrimSpace(a), "/") == strings.TrimRight(strings.TrimSpace(b), "/")
}
