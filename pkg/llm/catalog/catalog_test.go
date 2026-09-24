package catalog

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCatalogDefaultsAndHotReload(t *testing.T) {
	providers, err := Load("")
	if err != nil || len(providers) != 13 {
		t.Fatalf("defaults: %v, %d", err, len(providers))
	}
	path := filepath.Join(t.TempDir(), "providers.json")
	for _, name := range []string{"First", "Updated"} {
		body := `[{"id":"local","name":"` + name + `","protocol":"openai_compatible","baseUrl":"http://localhost:9000/v1"}]`
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
		loaded, err := Load(path)
		if err != nil || len(loaded) != 1 || loaded[0].Name != name {
			t.Fatalf("reload: %v, %+v", err, loaded)
		}
	}
}

func TestCatalogRejectsInvalidConfiguration(t *testing.T) {
	for _, body := range []string{
		`invalid`, `[]`,
		`[{"id":"x","name":"X","protocol":"unknown"}]`,
		`[{"id":"x","name":"X","protocol":"openai"},{"id":"x","name":"Y","protocol":"openai"}]`,
		`[{"id":"x","name":"X","protocol":"openai_compatible","baseUrl":"file:///tmp/file"}]`,
		`[{"id":"x","name":"X","protocol":"openai","docsUrl":"javascript:alert(1)"}]`,
	} {
		path := filepath.Join(t.TempDir(), "providers.json")
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(path); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}
