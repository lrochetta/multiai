package assets

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const expectedProfileCount = 37

func TestExtractProfilesCount(t *testing.T) {
	dir := t.TempDir()
	n, err := ExtractProfiles(dir)
	if err != nil {
		t.Fatalf("ExtractProfiles: %v", err)
	}
	if n != expectedProfileCount {
		t.Errorf("wrote %d files, want %d", n, expectedProfileCount)
	}
	matches, err := filepath.Glob(filepath.Join(dir, "*.env"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(matches) != expectedProfileCount {
		t.Errorf("found %d .env files on disk, want %d", len(matches), expectedProfileCount)
	}
	for _, m := range matches {
		info, err := os.Stat(m)
		if err != nil {
			t.Fatalf("stat %s: %v", m, err)
		}
		if info.Size() == 0 {
			t.Errorf("%s is empty", m)
		}
	}
}

func TestExtractProfilesCreatesDestDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "profiles")
	if _, err := ExtractProfiles(dir); err != nil {
		t.Fatalf("ExtractProfiles into non-existing dir: %v", err)
	}
}

func TestExtractProfilesDoesNotOverwrite(t *testing.T) {
	dir := t.TempDir()
	if _, err := ExtractProfiles(dir); err != nil {
		t.Fatalf("first extract: %v", err)
	}

	// Simulate a user edit.
	modified := filepath.Join(dir, "10-claude-anthropic-api.env")
	userContent := []byte("PROFILE_ID=user-modified\n")
	if err := os.WriteFile(modified, userContent, 0o600); err != nil {
		t.Fatalf("write user file: %v", err)
	}

	n, err := ExtractProfiles(dir)
	if err != nil {
		t.Fatalf("second extract: %v", err)
	}
	if n != 0 {
		t.Errorf("second extract wrote %d files, want 0", n)
	}

	got, err := os.ReadFile(modified)
	if err != nil {
		t.Fatalf("read user file: %v", err)
	}
	if string(got) != string(userContent) {
		t.Errorf("user modification was overwritten: got %q", got)
	}
}

// TestEmbeddedProfilesContainNoRealSecrets guards against a real API key
// (or a dangling credential-store sentinel) slipping into the embedded
// templates, which are committed to a public repository. Any variable whose
// name looks secret-bearing must hold a placeholder.
func TestEmbeddedProfilesContainNoRealSecrets(t *testing.T) {
	secretName := regexp.MustCompile(`KEY|TOKEN|SECRET|AUTH|PASSWORD|CREDENTIAL`)
	// Metadata keys that mention secrets without carrying one.
	exempt := map[string]bool{
		"REQUIRED_SECRETS":  true,
		"SKIP_SECRET_CHECK": true,
	}
	// Shapes of real credentials (Anthropic, OpenAI, generic long tokens).
	keyShape := regexp.MustCompile(`sk-|api03|Bearer |[A-Za-z0-9]{30,}`)
	// Pure %VAR% indirection (PS-style), e.g. ANTHROPIC_AUTH_TOKEN=%REQUESTY_API_KEY%.
	// Not a secret by itself, but the referenced variable must be defined in the
	// same file and hold a placeholder, so the indirection cannot launder a key.
	varRef := regexp.MustCompile(`^%([A-Za-z_][A-Za-z0-9_]*)%$`)

	err := fs.WalkDir(Profiles, "profiles", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := Profiles.ReadFile(path)
		if err != nil {
			return err
		}
		vars := parseVars(string(data))
		for i, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			eq := strings.Index(line, "=")
			if eq < 1 {
				continue
			}
			name := strings.TrimSpace(line[:eq])
			value := strings.TrimSpace(line[eq+1:])
			if exempt[name] || !secretName.MatchString(name) {
				continue
			}
			if m := varRef.FindStringSubmatch(value); m != nil {
				target, ok := vars[m[1]]
				if !ok {
					t.Errorf("%s:%d: %s references %%%s%% which is not defined in the file", path, i+1, name, m[1])
				} else if !isTemplatePlaceholder(target) {
					t.Errorf("%s:%d: %s references %%%s%% whose value %q is not a placeholder", path, i+1, name, m[1], target)
				}
				continue
			}
			if !isTemplatePlaceholder(value) {
				t.Errorf("%s:%d: %s has a non-placeholder value %q", path, i+1, name, value)
			}
		}
		// Belt and braces: no key-shaped string anywhere in the file.
		if loc := keyShape.FindString(string(data)); loc != "" {
			t.Errorf("%s: contains key-shaped string %q", path, loc)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk embedded profiles: %v", err)
	}
}

// TestEmbeddedProfilesHaveUniqueShortcuts guards the implicit contract that
// SHORTCUT values are the user-facing profile IDs: a duplicate would make one
// of the two profiles unreachable from the launcher.
func TestEmbeddedProfilesHaveUniqueShortcuts(t *testing.T) {
	seen := map[string]string{} // shortcut -> first file
	count := 0
	err := fs.WalkDir(Profiles, "profiles", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := Profiles.ReadFile(path)
		if err != nil {
			return err
		}
		count++
		shortcut := parseVars(string(data))["SHORTCUT"]
		if shortcut == "" {
			t.Errorf("%s: missing SHORTCUT", path)
			return nil
		}
		if prev, dup := seen[shortcut]; dup {
			t.Errorf("%s: SHORTCUT %q already used by %s", path, shortcut, prev)
		}
		seen[shortcut] = path
		return nil
	})
	if err != nil {
		t.Fatalf("walk embedded profiles: %v", err)
	}
	if count != expectedProfileCount {
		t.Errorf("walked %d embedded profiles, want %d", count, expectedProfileCount)
	}
}

// parseVars extracts NAME=value pairs from a .env template, ignoring blank
// lines and comments. Later duplicates win, matching loader semantics.
func parseVars(data string) map[string]string {
	vars := map[string]string{}
	for _, line := range strings.Split(data, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		eq := strings.Index(line, "=")
		if eq < 1 {
			continue
		}
		vars[strings.TrimSpace(line[:eq])] = strings.TrimSpace(line[eq+1:])
	}
	return vars
}

// isTemplatePlaceholder mirrors dotenv.IsPlaceholder, without importing
// internal packages. The credential-store sentinel is deliberately NOT
// accepted: a template shipping "__MULTIAI_CREDSTORE__" would embed a
// dangling reference in every released binary and break first-run.
func isTemplatePlaceholder(v string) bool {
	if v == "" {
		return true
	}
	lower := strings.ToLower(v)
	for _, p := range []string{"paste_", "your_", "xxx", "todo", "replace_me", "change_me"} {
		if strings.HasPrefix(lower, p) {
			return true
		}
	}
	return strings.HasSuffix(lower, "_here")
}
