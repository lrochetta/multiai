package openrouter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// loadFixture parses testdata/models.json (same envelope as the live API).
func loadFixture(t *testing.T) []ModelInfo {
	t.Helper()
	data, err := os.ReadFile("testdata/models.json")
	if err != nil {
		t.Fatalf("fixture: %v", err)
	}
	var envelope struct {
		Data []ModelInfo `json:"data"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatalf("fixture JSON: %v", err)
	}
	if len(envelope.Data) == 0 {
		t.Fatal("fixture is empty")
	}
	return envelope.Data
}

// setAPIBase points the client at a test server for the duration of a test.
func setAPIBase(t *testing.T, url string) {
	t.Helper()
	old := apiBase
	apiBase = url
	t.Cleanup(func() { apiBase = old })
}

// newFixtureServer serves testdata/models.json on /models and records the
// last request for header assertions.
func newFixtureServer(t *testing.T, lastReq *http.Request) *httptest.Server {
	t.Helper()
	data, err := os.ReadFile("testdata/models.json")
	if err != nil {
		t.Fatalf("fixture: %v", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if lastReq != nil {
			*lastReq = *r.Clone(r.Context())
		}
		if r.URL.Path != "/models" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// deadServer returns a base URL that refuses connections immediately.
func deadServer(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close()
	return url
}

func TestFetchModelsOK(t *testing.T) {
	var lastReq http.Request
	srv := newFixtureServer(t, &lastReq)
	setAPIBase(t, srv.URL)

	ctx := context.Background()
	models, err := FetchModels(ctx, "test-key")
	if err != nil {
		t.Fatalf("FetchModels: %v", err)
	}
	if len(models) != 8 {
		t.Fatalf("got %d models, want 8", len(models))
	}
	if got := lastReq.Header.Get("User-Agent"); got != "multiai" {
		t.Errorf("User-Agent = %q, want %q", got, "multiai")
	}
	if got := lastReq.Header.Get("Authorization"); got != "Bearer test-key" {
		t.Errorf("Authorization = %q, want bearer token", got)
	}
	var found bool
	for _, m := range models {
		if m.ID == "anthropic/claude-sonnet-4.6" {
			found = true
			if m.ContextLength != 1000000 {
				t.Errorf("context_length = %d, want 1000000", m.ContextLength)
			}
			if m.Pricing.Prompt != "0.000003" {
				t.Errorf("pricing.prompt = %q", m.Pricing.Prompt)
			}
			if !m.TopProvider.IsModerated {
				t.Error("top_provider.is_moderated should be true")
			}
		}
	}
	if !found {
		t.Error("claude-sonnet-4.6 missing from parsed models")
	}
}

func TestFetchModelsNoAuthHeaderWithoutKey(t *testing.T) {
	var lastReq http.Request
	srv := newFixtureServer(t, &lastReq)
	setAPIBase(t, srv.URL)

	ctx := context.Background()
	if _, err := FetchModels(ctx, ""); err != nil {
		t.Fatalf("FetchModels: %v", err)
	}
	if got := lastReq.Header.Get("Authorization"); got != "" {
		t.Errorf("Authorization = %q, want empty", got)
	}
}

func TestFetchModelsHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	setAPIBase(t, srv.URL)

	ctx := context.Background()
	if _, err := FetchModels(ctx, ""); err == nil || !strings.Contains(err.Error(), "500") {
		t.Fatalf("want HTTP 500 error, got %v", err)
	}
}

func TestFetchModelsBadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("{not json"))
	}))
	t.Cleanup(srv.Close)
	setAPIBase(t, srv.URL)

	ctx := context.Background()
	if _, err := FetchModels(ctx, ""); err == nil || !strings.Contains(err.Error(), "illisible") {
		t.Fatalf("want unreadable-response error, got %v", err)
	}
}

func TestFetchModelsResponseTooLarge(t *testing.T) {
	oldMax := maxResponseBytes
	maxResponseBytes = 64
	t.Cleanup(func() { maxResponseBytes = oldMax })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":[` + strings.Repeat(`{"id":"x"},`, 50) + `{"id":"y"}]}`))
	}))
	t.Cleanup(srv.Close)
	setAPIBase(t, srv.URL)

	ctx := context.Background()
	if _, err := FetchModels(ctx, ""); err == nil || !strings.Contains(err.Error(), "volumineuse") {
		t.Fatalf("want size-cap error, got %v", err)
	}
}

func TestFetchModelsConnectionRefused(t *testing.T) {
	setAPIBase(t, deadServer(t))
	ctx := context.Background()
	if _, err := FetchModels(ctx, ""); err == nil || !strings.Contains(err.Error(), "inaccessible") {
		t.Fatalf("want unreachable-API error, got %v", err)
	}
}
