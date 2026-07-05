package openrouter

import (
	"strings"
	"testing"
)

func ids(models []ModelInfo) []string {
	out := make([]string, len(models))
	for i, m := range models {
		out[i] = m.ID
	}
	return out
}

func TestTopSorting(t *testing.T) {
	models := loadFixture(t)
	tests := []struct {
		name    string
		sortKey string
		n       int
		first   string
		last    string
	}{
		{"recent", SortRecent, 0, "openrouter/owl-alpha", "nvidia/nemotron-3-ultra"},
		{"default is recent", "", 0, "openrouter/owl-alpha", "nvidia/nemotron-3-ultra"},
		{"price ascending, free first", SortPrice, 0, "nvidia/nemotron-3-ultra", "x-ai/grok-4.3"},
		{"context descending", SortContext, 0, "google/gemini-3.5-flash", "nvidia/nemotron-3-ultra"},
		{"name ascending", SortName, 0, "anthropic/claude-sonnet-4.6", "x-ai/grok-4.3"},
		{"truncation", SortRecent, 3, "openrouter/owl-alpha", "openai/gpt-5.5"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Top(models, tt.sortKey, tt.n)
			if err != nil {
				t.Fatalf("Top: %v", err)
			}
			if tt.n > 0 && len(got) != tt.n {
				t.Fatalf("len = %d, want %d", len(got), tt.n)
			}
			if got[0].ID != tt.first {
				t.Errorf("first = %s, want %s (order: %v)", got[0].ID, tt.first, ids(got))
			}
			if got[len(got)-1].ID != tt.last {
				t.Errorf("last = %s, want %s (order: %v)", got[len(got)-1].ID, tt.last, ids(got))
			}
		})
	}
}

func TestTopUnknownSortKey(t *testing.T) {
	if _, err := Top(loadFixture(t), "usage", 5); err == nil || !strings.Contains(err.Error(), "tri inconnu") {
		t.Fatalf("want unknown-sort error, got %v", err)
	}
}

func TestTopDoesNotMutateInput(t *testing.T) {
	models := loadFixture(t)
	firstBefore := models[0].ID
	if _, err := Top(models, SortName, 0); err != nil {
		t.Fatal(err)
	}
	if models[0].ID != firstBefore {
		t.Fatal("Top must sort a copy, not the caller's slice")
	}
}

func TestSearch(t *testing.T) {
	models := loadFixture(t)
	tests := []struct {
		name  string
		query string
		want  []string
	}{
		{"single term id", "deepseek", []string{"deepseek/deepseek-v4-pro"}},
		{"case insensitive name", "GEMINI", []string{"google/gemini-3.5-flash"}},
		{"description match", "cloaked", []string{"openrouter/owl-alpha"}},
		{"multi-term AND", "free reasoning", []string{"nvidia/nemotron-3-ultra"}},
		{"no match", "nonexistent-model-xyz", nil},
		{"empty query", "   ", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Search(models, tt.query)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", ids(got), tt.want)
			}
			for i := range tt.want {
				if got[i].ID != tt.want[i] {
					t.Errorf("result[%d] = %s, want %s", i, got[i].ID, tt.want[i])
				}
			}
		})
	}
}

func TestFindModel(t *testing.T) {
	models := loadFixture(t)
	tests := []struct {
		name    string
		query   string
		wantID  string
		wantErr string
	}{
		{"exact id", "openai/gpt-5.5", "openai/gpt-5.5", ""},
		{"exact id case-insensitive", "OpenAI/GPT-5.5", "openai/gpt-5.5", ""},
		{"unique substring", "grok", "x-ai/grok-4.3", ""},
		{"unique name substring", "nemotron", "nvidia/nemotron-3-ultra", ""},
		{"not found", "does-not-exist", "", "introuvable"},
		{"vendor substring still unique", "deepseek", "deepseek/deepseek-v4-pro", ""},
		{"ambiguous multi", "e", "", "plusieurs"},
		{"empty", "  ", "", "vide"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FindModel(models, tt.query)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("want error containing %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("FindModel: %v", err)
			}
			if got.ID != tt.wantID {
				t.Errorf("got %s, want %s", got.ID, tt.wantID)
			}
		})
	}
}

func TestFilters(t *testing.T) {
	models := loadFixture(t)

	free := FilterFree(models)
	if got := ids(free); len(got) != 2 {
		t.Errorf("FilterFree = %v, want 2 free models", got)
	}

	multimodal := FilterModality(models, "image")
	if len(multimodal) != 3 {
		t.Errorf("FilterModality(image) = %v, want 3", ids(multimodal))
	}

	anthropic := FilterVendor(models, "Anthropic")
	if len(anthropic) != 1 || anthropic[0].ID != "anthropic/claude-sonnet-4.6" {
		t.Errorf("FilterVendor(Anthropic) = %v", ids(anthropic))
	}

	none := FilterVendor(models, "unknown-vendor")
	if len(none) != 0 {
		t.Errorf("FilterVendor(unknown) = %v, want empty", ids(none))
	}
}

func TestFormatPricePerMTok(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"0.000003", "3.00"},
		{"0.0000006", "0.60"},
		{"0", "gratuit"},
		{"", "n/d"},
		{"-1", "n/d"},
		{"abc", "n/d"},
	}
	for _, tt := range tests {
		if got := FormatPricePerMTok(tt.in); got != tt.want {
			t.Errorf("FormatPricePerMTok(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestFormatContext(t *testing.T) {
	tests := []struct {
		in   int
		want string
	}{
		{0, "n/d"},
		{512, "512"},
		{131072, "131k"},
		{1048576, "1048k"},
	}
	for _, tt := range tests {
		if got := formatContext(tt.in); got != tt.want {
			t.Errorf("formatContext(%d) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestRenderModelTable(t *testing.T) {
	var sb strings.Builder
	RenderModelTable(&sb, loadFixture(t)[:2])
	out := sb.String()
	for _, want := range []string{"ID", "CTX", "IN $/M", "anthropic/claude-sonnet-4.6", "3.00", "1000k"} {
		if !strings.Contains(out, want) {
			t.Errorf("table output missing %q:\n%s", want, out)
		}
	}
}

func TestRenderComparison(t *testing.T) {
	models := loadFixture(t)
	a, _ := FindModel(models, "anthropic/claude-sonnet-4.6")
	b, _ := FindModel(models, "nvidia/nemotron-3-ultra")
	var sb strings.Builder
	RenderComparison(&sb, *a, *b)
	out := sb.String()
	for _, want := range []string{"Prix entree ($/M)", "gratuit", "3.00", "Moderation", "oui", "non", "Tokenizer"} {
		if !strings.Contains(out, want) {
			t.Errorf("comparison output missing %q:\n%s", want, out)
		}
	}
}
