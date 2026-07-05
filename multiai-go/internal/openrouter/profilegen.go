package openrouter

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/lrochetta/multiai/internal/fsutil"
)

// ErrProfileExists is returned by CreateProfile when the target .env already
// exists and overwrite is false. The PowerShell reference overwrote silently
// (two display names can collapse to the same 12-char shortcut); the Go port
// deliberately refuses and lets the caller confirm.
var ErrProfileExists = errors.New("le fichier de profil existe deja")

var (
	// shortcutStrip removes everything but ASCII alphanumerics, like the PS
	// `-replace '[^a-zA-Z0-9]',''` (New-OpenRouterProfile L944).
	shortcutStrip = regexp.MustCompile(`[^a-zA-Z0-9]`)
	// slugPattern validates OpenRouter model slugs ("vendor/model"). The PS
	// reference accepted anything; the Go port validates to avoid generating
	// broken profiles.
	slugPattern = regexp.MustCompile(`^[\w.-]+/[\w.:-]+$`)
)

// ProfileSpec describes a dynamic OpenRouter launch profile to generate.
type ProfileSpec struct {
	DisplayName string // e.g. "DeepSeek V4 Pro"
	ModelSlug   string // e.g. "deepseek/deepseek-v4-pro"
	Tool        string // claude | codex | opencode
}

// Shortcut derives the profile shortcut from a display name, PS-compatible:
// "or-" + alphanumerics lowercased, truncated to 12 characters total.
func Shortcut(displayName string) string {
	s := "or-" + strings.ToLower(shortcutStrip.ReplaceAllString(displayName, ""))
	if len(s) > 12 {
		s = s[:12]
	}
	return s
}

// ProfileFileName returns the .env file name for a display name ("99-<shortcut>.env").
func ProfileFileName(displayName string) string {
	return "99-" + Shortcut(displayName) + ".env"
}

// ActiveProfilesDir resolves the profiles directory the same way the binary
// does for user installs: MULTIAI_PROFILES_DIR when set, otherwise
// <user config dir>/multiai/profiles. It does not create the directory.
func ActiveProfilesDir() (string, error) {
	if dir := os.Getenv("MULTIAI_PROFILES_DIR"); dir != "" {
		return dir, nil
	}
	cfg, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("dossier de configuration utilisateur introuvable: %w", err)
	}
	return filepath.Join(cfg, "multiai", "profiles"), nil
}

// validate checks a spec before rendering. Control characters in the display
// name are rejected (they would inject lines into the generated .env).
func (s ProfileSpec) validate() error {
	name := strings.TrimSpace(s.DisplayName)
	if name == "" {
		return errors.New("nom de modele vide")
	}
	for _, r := range name {
		if r < 0x20 || r == 0x7f {
			return errors.New("nom de modele invalide (caractere de controle)")
		}
	}
	if shortcutStrip.ReplaceAllString(name, "") == "" {
		return fmt.Errorf("nom de modele invalide : %q ne contient aucun caractere alphanumerique", s.DisplayName)
	}
	if !slugPattern.MatchString(s.ModelSlug) {
		return fmt.Errorf("slug OpenRouter invalide : %q (attendu : fournisseur/modele, ex. deepseek/deepseek-v4-pro)", s.ModelSlug)
	}
	switch s.Tool {
	case "claude", "codex", "opencode":
	default:
		return fmt.Errorf("outil inconnu %q (valides : claude, codex, opencode)", s.Tool)
	}
	return nil
}

// Render produces the .env content for a dynamic profile, byte-for-byte
// aligned with the PowerShell generator (New-OpenRouterProfile L942-982):
// CRLF line endings, trailing newline, placeholder key, %VAR% reference to
// OPENROUTER_API_KEY resolved at launch by the router's env expansion.
func Render(spec ProfileSpec) (string, error) {
	if err := spec.validate(); err != nil {
		return "", err
	}
	sc := Shortcut(spec.DisplayName)
	lines := []string{
		"PROFILE_ID=" + sc,
		"SHORTCUT=" + sc,
		"TOOL=" + spec.Tool,
		"TOOL_LABEL=" + spec.Tool,
		"DISPLAY_NAME=" + strings.TrimSpace(spec.DisplayName) + " (via OR)",
		"DESCRIPTION=OpenRouter: " + spec.ModelSlug,
		"ORDER=50",
		"COMMAND=" + spec.Tool,
		"CLEAR_ENV=true",
		"REQUIRED_SECRETS=OPENROUTER_API_KEY",
		"OPENROUTER_API_KEY=PASTE_OPENROUTER_API_KEY_HERE",
	}
	switch spec.Tool {
	case "claude":
		// /api without /v1: Claude Code appends /v1/messages itself.
		lines = append(lines,
			"ANTHROPIC_AUTH_TOKEN=%OPENROUTER_API_KEY%",
			"ANTHROPIC_BASE_URL=https://openrouter.ai/api",
			"ANTHROPIC_MODEL="+spec.ModelSlug,
			"ANTHROPIC_API_KEY=",
		)
	case "codex":
		lines = append(lines,
			"OPENAI_API_KEY=%OPENROUTER_API_KEY%",
			"OPENAI_BASE_URL=https://openrouter.ai/api/v1",
		)
	case "opencode":
		// OPENROUTER_API_KEY is the native variable: nothing to add.
	}
	return strings.Join(lines, "\r\n") + "\r\n", nil
}

// CreateProfile writes the generated .env into dir (created when missing).
// When the target file exists and overwrite is false it returns the path
// together with ErrProfileExists (wrapped), so callers can ask the user.
func CreateProfile(dir string, spec ProfileSpec, overwrite bool) (string, error) {
	content, err := Render(spec)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creation du dossier profils impossible: %w", err)
	}
	path := filepath.Join(dir, ProfileFileName(spec.DisplayName))
	if !overwrite {
		if _, statErr := os.Stat(path); statErr == nil {
			return path, fmt.Errorf("%w : %s", ErrProfileExists, path)
		}
	}
	// Atomic overwrite: a failed/interrupted write must not destroy a valid
	// existing profile (the caller may have confirmed an overwrite).
	if err := fsutil.WriteFileAtomic(path, []byte(content), 0o600); err != nil {
		return "", fmt.Errorf("ecriture du profil impossible: %w", err)
	}
	return path, nil
}
