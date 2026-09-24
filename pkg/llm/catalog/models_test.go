package catalog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/mayswind/financially_free/pkg/settings"
)

func TestListModelsRequestAndNormalization(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/v1/models" || r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("unexpected model request")
		}
		w.Write([]byte(`{"data":[{"id":"z"},{"id":"a"},{"id":"a"},{"id":""},{"id":"embedding","type":"embeddings"}]}`))
	}))
	defer server.Close()
	result, err := ListModels(context.Background(), &settings.LLMConfig{LLMProvider: "openai_compatible", OpenAICompatibleBaseURL: server.URL + "/v1/", OpenAICompatibleAPIKey: "test-key", LargeLanguageModelAPIProxy: "none"})
	if err != nil || !reflect.DeepEqual(result, []string{"a", "z"}) {
		t.Fatalf("%v, %v", result, err)
	}
}

func TestModelFailuresAndRedirects(t *testing.T) {
	destinationCalled := false
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { destinationCalled = true }))
	defer destination.Close()
	for _, status := range []int{401, 404, 429, 500, 302, 200} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Location", destination.URL)
			w.WriteHeader(status)
			w.Write([]byte(`{"error":"no models"}`))
		}))
		_, err := ListModels(context.Background(), &settings.LLMConfig{LLMProvider: "openai_compatible", OpenAICompatibleBaseURL: server.URL, LargeLanguageModelAPIProxy: "none"})
		server.Close()
		if err == nil {
			t.Errorf("expected failure for %d", status)
		}
	}
	if destinationCalled {
		t.Error("followed credential-bearing redirect")
	}
}

func TestNativeModelLists(t *testing.T) {
	tests := []struct {
		protocol, body string
		want           []string
		next           string
	}{
		{"ollama", `{"models":[{"name":"qwen:latest"}]}`, []string{"qwen:latest"}, ""},
		{"google_ai", `{"models":[{"name":"models/gemini","supportedGenerationMethods":["generateContent"]},{"name":"models/embed","supportedGenerationMethods":["embedContent"]}],"nextPageToken":"next"}`, []string{"gemini"}, "next"},
		{"openai_compatible", `{"data":[]}`, []string{}, ""},
	}
	for _, tt := range tests {
		got, next, err := parseModels([]byte(tt.body), tt.protocol)
		if err != nil || !reflect.DeepEqual(got, tt.want) || next != tt.next {
			t.Errorf("%s: %v, %s, %v", tt.protocol, got, next, err)
		}
	}
}
