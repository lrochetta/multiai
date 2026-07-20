package openrouter

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestNvidiaShortcut(t *testing.T) {
	tests := []struct{ in, want string }{
		{"GLM 5.2", "nv-glm52"},
		{"DeepSeek V4 Flash", "nv-deepseekv"}, // truncated to 12
		{"a", "nv-a"},
	}
	for _, tt := range tests {
		if got := NvidiaShortcut(tt.in); got != tt.want {
			t.Errorf("NvidiaShortcut(%q) = %q, want %q", tt.in, got, tt.want)
		}
		if len(NvidiaShortcut(tt.in)) > 12 {
			t.Errorf("NvidiaShortcut(%q) longer than 12", tt.in)
		}
	}
}

func TestNvidiaProfileFileName(t *testing.T) {
	if got := NvidiaProfileFileName("GLM 5.2"); got != "98-nv-glm52.env" {
		t.Errorf("NvidiaProfileFileName = %q, want 98-nv-glm52.env", got)
	}
}

func TestRenderNvidiaClaude(t *testing.T) {
	content, err := RenderNvidia(ProfileSpec{DisplayName: "GLM 5.2", ModelSlug: "z-ai/glm-5.2", Tool: "claude"})
	if err != nil {
		t.Fatalf("RenderNvidia: %v", err)
	}
	for _, want := range []string{
		"SHORTCUT=nv-glm52",
		"REQUIRED_SECRETS=NVIDIA_API_KEY",
		"NVIDIA_API_KEY=PASTE_NVIDIA_API_KEY_HERE",
		"BRIDGE=anthropic-openai",
		"BRIDGE_TARGET=https://integrate.api.nvidia.com/v1",
		"BRIDGE_KEY_VAR=NVIDIA_API_KEY",
		"ANTHROPIC_AUTH_TOKEN=%NVIDIA_API_KEY%",
		"ANTHROPIC_MODEL=z-ai/glm-5.2",
		"ANTHROPIC_API_KEY=",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("claude profile missing %q:\n%s", want, content)
		}
	}
}

func TestRenderNvidiaCodex(t *testing.T) {
	content, err := RenderNvidia(ProfileSpec{DisplayName: "GLM 5.2", ModelSlug: "z-ai/glm-5.2", Tool: "codex"})
	if err != nil {
		t.Fatalf("RenderNvidia: %v", err)
	}
	for _, want := range []string{
		"-c model=z-ai/glm-5.2",
		"-c model_providers.nvidia.base_url=" + nvidiaBridgeURL,
		"-c model_providers.nvidia.wire_api=responses",
		"-c model_providers.nvidia.env_key=NVIDIA_API_KEY",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("codex profile missing %q:\n%s", want, content)
		}
	}
}

func TestRenderNvidiaOpencode(t *testing.T) {
	content, err := RenderNvidia(ProfileSpec{DisplayName: "GLM 5.2", ModelSlug: "z-ai/glm-5.2", Tool: "opencode"})
	if err != nil {
		t.Fatalf("RenderNvidia: %v", err)
	}
	var cfgLine string
	for _, line := range strings.Split(content, "\r\n") {
		if strings.HasPrefix(line, "OPENCODE_CONFIG_CONTENT=") {
			cfgLine = strings.TrimPrefix(line, "OPENCODE_CONFIG_CONTENT=")
		}
	}
	if cfgLine == "" {
		t.Fatalf("opencode profile has no OPENCODE_CONFIG_CONTENT:\n%s", content)
	}
	var cfg struct {
		Model    string `json:"model"`
		Provider map[string]struct {
			NPM     string `json:"npm"`
			Options struct {
				BaseURL string `json:"baseURL"`
				APIKey  string `json:"apiKey"`
			} `json:"options"`
		} `json:"provider"`
	}
	if err := json.Unmarshal([]byte(cfgLine), &cfg); err != nil {
		t.Fatalf("OPENCODE_CONFIG_CONTENT is not valid JSON: %v\n%s", err, cfgLine)
	}
	if cfg.Model != "nvidia/z-ai/glm-5.2" {
		t.Errorf("model = %q, want nvidia/z-ai/glm-5.2", cfg.Model)
	}
	nv, ok := cfg.Provider["nvidia"]
	if !ok {
		t.Fatal("provider nvidia missing")
	}
	if nv.NPM != "@ai-sdk/openai-compatible" {
		t.Errorf("npm = %q", nv.NPM)
	}
	if nv.Options.APIKey != "{env:NVIDIA_API_KEY}" {
		t.Errorf("apiKey = %q, want {env:NVIDIA_API_KEY}", nv.Options.APIKey)
	}
}

func TestRenderNvidiaInvalidSlug(t *testing.T) {
	if _, err := RenderNvidia(ProfileSpec{DisplayName: "X", ModelSlug: "pas-un-slug", Tool: "claude"}); err == nil {
		t.Error("expected error for invalid slug")
	}
}

func TestSortNvidia(t *testing.T) {
	models := []ModelInfo{
		{ID: "z-ai/glm-5.2", OwnedBy: "z-ai"},
		{ID: "meta/llama-4-maverick-17b-128e-instruct", OwnedBy: "meta"},
		{ID: "deepseek-ai/deepseek-v4-pro", OwnedBy: "deepseek-ai"},
	}
	byName, err := SortNvidia(models, "")
	if err != nil {
		t.Fatalf("SortNvidia: %v", err)
	}
	if byName[0].ID != "deepseek-ai/deepseek-v4-pro" || byName[2].ID != "z-ai/glm-5.2" {
		t.Errorf("nom sort wrong: %v", byName)
	}
	byOwner, err := SortNvidia(models, "editeur")
	if err != nil {
		t.Fatalf("SortNvidia editeur: %v", err)
	}
	if byOwner[0].OwnedBy != "deepseek-ai" || byOwner[1].OwnedBy != "meta" {
		t.Errorf("editeur sort wrong: %v", byOwner)
	}
	if _, err := SortNvidia(models, "prix"); err == nil {
		t.Error("expected error for unknown sort key")
	}
}

func TestFetchNvidiaModels(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			http.NotFound(w, r)
			return
		}
		gotAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"data":[{"id":"z-ai/glm-5.2","object":"model","created":735790403,"owned_by":"z-ai"}]}`))
	}))
	defer srv.Close()

	old := nvidiaAPIBase
	nvidiaAPIBase = srv.URL
	defer func() { nvidiaAPIBase = old }()

	models, err := FetchNvidiaModels(context.Background(), "nvapi-test")
	if err != nil {
		t.Fatalf("FetchNvidiaModels: %v", err)
	}
	if len(models) != 1 || models[0].ID != "z-ai/glm-5.2" || models[0].OwnedBy != "z-ai" {
		t.Errorf("unexpected models: %+v", models)
	}
	if gotAuth != "Bearer nvapi-test" {
		t.Errorf("Authorization = %q, want Bearer nvapi-test", gotAuth)
	}
}

func TestCreateNvidiaProfileOverwriteProtocol(t *testing.T) {
	dir := t.TempDir()
	spec := ProfileSpec{DisplayName: "GLM 5.2", ModelSlug: "z-ai/glm-5.2", Tool: "opencode"}
	path, err := CreateNvidiaProfile(dir, spec, false)
	if err != nil {
		t.Fatalf("CreateNvidiaProfile: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("profile not written: %v", err)
	}
	if _, err := CreateNvidiaProfile(dir, spec, false); err == nil {
		t.Error("expected ErrProfileExists on second create")
	}
	if _, err := CreateNvidiaProfile(dir, spec, true); err != nil {
		t.Errorf("overwrite=true must succeed: %v", err)
	}
}

func TestCreateNvidiaProfileShortcutCollision(t *testing.T) {
	dir := t.TempDir()
	// Simulate the bundled 84-claude-nvidia.env which claims nv-cc.
	bundled := "PROFILE_ID=claude-nvidia\r\nSHORTCUT=nv-cc\r\nTOOL=claude\r\n"
	if err := os.WriteFile(dir+string(os.PathSeparator)+"84-claude-nvidia.env", []byte(bundled), 0o600); err != nil {
		t.Fatalf("seed bundled profile: %v", err)
	}
	// "CC" derives shortcut nv-cc -> must be refused with a clear error.
	_, err := CreateNvidiaProfile(dir, ProfileSpec{DisplayName: "CC", ModelSlug: "z-ai/glm-5.2", Tool: "opencode"}, false)
	if err == nil || !strings.Contains(err.Error(), "deja pris") {
		t.Fatalf("expected shortcut collision error, got %v", err)
	}
	// Overwriting the SAME generated file must stay allowed (self is ignored).
	if _, err := CreateNvidiaProfile(dir, ProfileSpec{DisplayName: "GLM 5.2", ModelSlug: "z-ai/glm-5.2", Tool: "opencode"}, false); err != nil {
		t.Fatalf("unrelated create must pass: %v", err)
	}
	if _, err := CreateNvidiaProfile(dir, ProfileSpec{DisplayName: "GLM 5.2", ModelSlug: "z-ai/glm-5.2", Tool: "opencode"}, true); err != nil {
		t.Fatalf("self overwrite must pass: %v", err)
	}
}

func TestRenderNvidiaSlugErrorWording(t *testing.T) {
	_, err := RenderNvidia(ProfileSpec{DisplayName: "X", ModelSlug: "glm5.2", Tool: "claude"})
	if err == nil || !strings.Contains(err.Error(), "slug NVIDIA invalide") {
		t.Fatalf("expected NVIDIA-worded slug error, got %v", err)
	}
}

func TestRunNvidiaMenuExitAndInvalid(t *testing.T) {
	t.Setenv("MULTIAI_PROFILES_DIR", t.TempDir())
	var out bytes.Buffer
	in := bufio.NewReader(strings.NewReader("x\n0\n"))
	if err := runNvidiaMenu(in, &out); err != nil {
		t.Fatalf("runNvidiaMenu: %v", err)
	}
	s := out.String()
	if !strings.Contains(s, "NVIDIA build.nvidia.com") {
		t.Errorf("menu header missing:\n%s", s)
	}
	if !strings.Contains(s, "Choix invalide") {
		t.Errorf("invalid choice warning missing:\n%s", s)
	}
	if !strings.Contains(s, nvidiaKeysURL) {
		t.Errorf("key generation URL missing:\n%s", s)
	}
}

func TestRunNvidiaMenuCreateProfile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MULTIAI_PROFILES_DIR", dir)
	var out bytes.Buffer
	in := bufio.NewReader(strings.NewReader("3\nGLM 5.2\nz-ai/glm-5.2\nopencode\n0\n"))
	if err := runNvidiaMenu(in, &out); err != nil {
		t.Fatalf("runNvidiaMenu: %v", err)
	}
	if !strings.Contains(out.String(), "[OK] Profil cree") {
		t.Fatalf("profile creation not confirmed:\n%s", out.String())
	}
	if _, err := os.Stat(dir + string(os.PathSeparator) + "98-nv-glm52.env"); err != nil {
		t.Errorf("expected profile file: %v", err)
	}
}
