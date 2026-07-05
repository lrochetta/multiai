package openrouter

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// runScriptedMenu drives runMenu with scripted stdin lines and captures output.
func runScriptedMenu(t *testing.T, input string) string {
	t.Helper()
	var out strings.Builder
	if err := runMenu(bufio.NewReader(strings.NewReader(input)), &out); err != nil {
		t.Fatalf("runMenu: %v", err)
	}
	return out.String()
}

func TestMenuExitImmediately(t *testing.T) {
	t.Setenv("MULTIAI_PROFILES_DIR", t.TempDir())
	out := runScriptedMenu(t, "0\n")
	if !strings.Contains(out, "OpenRouter -- Decouvrir et ajouter des modeles") {
		t.Errorf("menu header missing:\n%s", out)
	}
}

func TestMenuEOFExitsCleanly(t *testing.T) {
	t.Setenv("MULTIAI_PROFILES_DIR", t.TempDir())
	_ = runScriptedMenu(t, "") // immediate EOF must not loop or panic
}

func TestMenuInvalidChoice(t *testing.T) {
	t.Setenv("MULTIAI_PROFILES_DIR", t.TempDir())
	out := runScriptedMenu(t, "9\n0\n")
	if !strings.Contains(out, "Choix invalide") {
		t.Errorf("invalid choice message missing:\n%s", out)
	}
}

func TestMenuCreateProfileDefaultTool(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MULTIAI_PROFILES_DIR", dir)

	out := runScriptedMenu(t, strings.Join([]string{
		"4",
		"DeepSeek V4 Pro",
		"deepseek/deepseek-v4-pro",
		"", // empty CLI -> claude
		"0",
	}, "\n")+"\n")

	if !strings.Contains(out, "[OK] Profil cree : DeepSeek V4 Pro [or-deepseekv] -> 99-or-deepseekv.env") {
		t.Errorf("success message missing:\n%s", out)
	}
	data, err := os.ReadFile(filepath.Join(dir, "99-or-deepseekv.env"))
	if err != nil {
		t.Fatalf("generated profile missing: %v", err)
	}
	if !strings.Contains(string(data), "ANTHROPIC_MODEL=deepseek/deepseek-v4-pro") {
		t.Errorf("claude profile content wrong:\n%s", data)
	}
}

func TestMenuCreateProfileAbortOnEmptyName(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MULTIAI_PROFILES_DIR", dir)

	runScriptedMenu(t, "4\n\n0\n") // empty name aborts the flow
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Errorf("no file should be created, got %d entries", len(entries))
	}
}

func TestMenuCreateProfileRejectsBadSlug(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MULTIAI_PROFILES_DIR", dir)

	out := runScriptedMenu(t, "4\nFoo\nbadslug\n\n0\n")
	if !strings.Contains(out, "slug OpenRouter invalide") {
		t.Errorf("slug validation message missing:\n%s", out)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Errorf("no file should be created, got %d entries", len(entries))
	}
}

func TestMenuCreateProfileOverwriteFlow(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MULTIAI_PROFILES_DIR", dir)
	spec := ProfileSpec{DisplayName: "DeepSeek V4 Pro", ModelSlug: "deepseek/old-model", Tool: "claude"}
	if _, err := CreateProfile(dir, spec, false); err != nil {
		t.Fatal(err)
	}

	// Refusal keeps the old file.
	out := runScriptedMenu(t, strings.Join([]string{
		"4", "DeepSeek V4 Pro", "deepseek/deepseek-v4-pro", "claude", "n",
		"0",
	}, "\n")+"\n")
	if !strings.Contains(out, "existe deja") || !strings.Contains(out, "Annule.") {
		t.Errorf("overwrite refusal flow broken:\n%s", out)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "99-or-deepseekv.env"))
	if !strings.Contains(string(data), "deepseek/old-model") {
		t.Errorf("file must be untouched after refusal:\n%s", data)
	}

	// Confirmation overwrites.
	out = runScriptedMenu(t, strings.Join([]string{
		"4", "DeepSeek V4 Pro", "deepseek/deepseek-v4-pro", "claude", "o",
		"0",
	}, "\n")+"\n")
	if !strings.Contains(out, "[OK] Profil cree") {
		t.Errorf("overwrite confirmation flow broken:\n%s", out)
	}
	data, _ = os.ReadFile(filepath.Join(dir, "99-or-deepseekv.env"))
	if !strings.Contains(string(data), "deepseek/deepseek-v4-pro") {
		t.Errorf("file must hold the new slug after overwrite:\n%s", data)
	}
}

func TestMenuTopOfflineFallsBackToEmbedded(t *testing.T) {
	t.Setenv("MULTIAI_PROFILES_DIR", t.TempDir())
	t.Setenv("MULTIAI_CACHE_DIR", t.TempDir())
	setAPIBase(t, deadServer(t)) // no network in tests

	out := runScriptedMenu(t, "1\n\n0\n") // default sort
	if !strings.Contains(out, "openrouter/fusion") {
		t.Errorf("embedded list should be shown:\n%s", out)
	}
	if !strings.Contains(out, "liste statique embarquee") {
		t.Errorf("degradation notice missing:\n%s", out)
	}
}

func TestMenuSearchAndCompareOnCache(t *testing.T) {
	t.Setenv("MULTIAI_PROFILES_DIR", t.TempDir())
	cacheDir := t.TempDir()
	t.Setenv("MULTIAI_CACHE_DIR", cacheDir)
	if err := SaveCache(loadFixture(t)); err != nil {
		t.Fatal(err)
	}
	setAPIBase(t, deadServer(t)) // fresh cache must be enough

	out := runScriptedMenu(t, strings.Join([]string{
		"2", "gemini",
		"3", "openai/gpt-5.5", "nemotron",
		"0",
	}, "\n")+"\n")

	if !strings.Contains(out, "google/gemini-3.5-flash") {
		t.Errorf("search result missing:\n%s", out)
	}
	if !strings.Contains(out, "Prix entree ($/M)") {
		t.Errorf("comparison table missing:\n%s", out)
	}
	if !strings.Contains(out, "cache local du") {
		t.Errorf("cache source label missing:\n%s", out)
	}
}
