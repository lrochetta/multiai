//go:build linux

package secret

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// ── Test helper process (fake secret-tool) ──────────────────────────────────

// TestSecretToolHelper is NOT a real test — it is invoked as a subprocess by
// the other tests to fake the "secret-tool" binary. The SECRET_TOOL_HELPER
// environment variable gates the behavior; when set, this function acts as a
// fake secret-tool and calls os.Exit.
func TestSecretToolHelper(t *testing.T) {
	if os.Getenv("SECRET_TOOL_HELPER") != "1" {
		t.Skip("not a secret-tool helper process")
	}

	// Find the "--" separator in os.Args (everything after it are the real args).
	realArgs := extractArgsAfterDash()
	if len(realArgs) == 0 {
		os.Exit(1)
		return
	}
	subcmd := realArgs[0]
	behavior := os.Getenv("SECRET_TOOL_BEHAVIOR")

	switch subcmd {
	case "store":
		data, err := io.ReadAll(os.Stdin)
		if err != nil || len(data) == 0 || behavior != "success" {
			os.Exit(1)
		}
		os.Exit(0)

	case "lookup":
		if behavior == "notfound" {
			fmt.Fprint(os.Stderr, "not found\n")
			os.Exit(1)
		}
		fmt.Print("test-secret-value")
		os.Exit(0)

	case "clear":
		if behavior != "success" {
			os.Exit(1)
		}
		os.Exit(0)

	case "search":
		if behavior == "empty" {
			os.Exit(0)
		}
		fmt.Print(`[/org/freedesktop/secrets/collection/login/1]
label = multiai
service = test-service
key = API_KEY
value = test-api-key-value

[/org/freedesktop/secrets/collection/login/2]
label = multiai
service = test-service
key = ANOTHER_KEY
value = another-value
`)
		os.Exit(0)

	case "--help":
		// Simulate secret-tool --help (used for availability check).
		fmt.Print("secret-tool 1.20\n")
		os.Exit(0)

	default:
		os.Exit(1)
	}
}

// extractArgsAfterDash returns os.Args elements after the first "--" separator.
func extractArgsAfterDash() []string {
	for i, a := range os.Args {
		if a == "--" && i+1 < len(os.Args) {
			return os.Args[i+1:]
		}
	}
	return nil
}

// ── Helper to mock execCommand ──────────────────────────────────────────────

// fakeSecretTool returns a replacement for execCommand that re-executes the test
// binary as the fake secret-tool, with the given behavior.
func fakeSecretTool(behavior string) func(string, ...string) *exec.Cmd {
	return func(name string, args ...string) *exec.Cmd {
		// Build args: -test.run to invoke the helper, then "--" as separator,
		// then the original command name and args.
		cmdArgs := []string{"-test.run=^TestSecretToolHelper$", "--", name}
		cmdArgs = append(cmdArgs, args...)
		cmd := exec.Command(os.Args[0], cmdArgs...)
		cmd.Env = append(os.Environ(),
			"SECRET_TOOL_HELPER=1",
			"SECRET_TOOL_BEHAVIOR="+behavior,
		)
		return cmd
	}
}

// ── Tests for libsecretStore operations ─────────────────────────────────────

func TestLibsecretStore_Get(t *testing.T) {
	orig := execCommand
	defer func() { execCommand = orig }()

	t.Run("success", func(t *testing.T) {
		execCommand = fakeSecretTool("success")
		s := &libsecretStore{}
		val, err := s.Get("test-service", "API_KEY")
		if err != nil {
			t.Fatal(err)
		}
		if val != "test-secret-value" {
			t.Errorf("Get = %q, want %q", val, "test-secret-value")
		}
	})

	t.Run("not found", func(t *testing.T) {
		execCommand = fakeSecretTool("notfound")
		s := &libsecretStore{}
		_, err := s.Get("test-service", "MISSING_KEY")
		if err == nil {
			t.Error("expected error for missing credential")
		}
		if !strings.Contains(err.Error(), "credential not found") {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestLibsecretStore_Set(t *testing.T) {
	orig := execCommand
	defer func() { execCommand = orig }()
	execCommand = fakeSecretTool("success")

	s := &libsecretStore{}
	if err := s.Set("test-service", "API_KEY", "secret-value"); err != nil {
		t.Errorf("Set: %v", err)
	}
}

func TestLibsecretStore_Delete(t *testing.T) {
	orig := execCommand
	defer func() { execCommand = orig }()
	execCommand = fakeSecretTool("success")

	s := &libsecretStore{}
	if err := s.Delete("test-service", "API_KEY"); err != nil {
		t.Errorf("Delete: %v", err)
	}
}

func TestLibsecretStore_List(t *testing.T) {
	orig := execCommand
	defer func() { execCommand = orig }()

	t.Run("with results", func(t *testing.T) {
		execCommand = fakeSecretTool("success")
		s := &libsecretStore{}
		creds, err := s.List("test-service")
		if err != nil {
			t.Fatal(err)
		}
		if len(creds) != 2 {
			t.Fatalf("List returned %d entries, want 2", len(creds))
		}
		if creds["API_KEY"] != "test-api-key-value" {
			t.Errorf("API_KEY = %q, want %q", creds["API_KEY"], "test-api-key-value")
		}
		if creds["ANOTHER_KEY"] != "another-value" {
			t.Errorf("ANOTHER_KEY = %q, want %q", creds["ANOTHER_KEY"], "another-value")
		}
	})

	t.Run("empty", func(t *testing.T) {
		execCommand = fakeSecretTool("empty")
		s := &libsecretStore{}
		creds, err := s.List("test-service")
		if err != nil {
			t.Fatal(err)
		}
		if len(creds) != 0 {
			t.Errorf("expected empty, got %d entries", len(creds))
		}
	})

	t.Run("search error", func(t *testing.T) {
		execCommand = fakeSecretTool("notfound")
		s := &libsecretStore{}
		creds, err := s.List("test-service")
		if err != nil {
			t.Fatal(err)
		}
		if len(creds) != 0 {
			t.Errorf("expected empty after search error, got %d entries", len(creds))
		}
	})
}

// ── Tests for newPlatformStore ──────────────────────────────────────────────

func TestNewPlatformStore_Unavailable(t *testing.T) {
	orig := secretToolLookPath
	defer func() { secretToolLookPath = orig }()

	secretToolLookPath = func(name string) (string, error) {
		return "", fmt.Errorf("not found: %s", name)
	}

	_, err := newPlatformStore()
	if err == nil {
		t.Fatal("newPlatformStore should return error when secret-tool is unavailable")
	}
	if !strings.Contains(err.Error(), "secret-tool not found") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// ── Tests for parseSecretToolSearch ─────────────────────────────────────────

func TestParseSecretToolSearch(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  map[string]string
	}{
		{
			name: "single entry",
			input: `[/org/freedesktop/secrets/collection/login/1]
label = multiai
service = test-svc
key = API_KEY
value = sk-ant-abc123
`,
			want: map[string]string{"API_KEY": "sk-ant-abc123"},
		},
		{
			name: "multiple entries",
			input: `[/1]
label = multiai
service = svc
key = K1
value = v1

[/2]
label = multiai
service = svc
key = K2
value = v2
`,
			want: map[string]string{"K1": "v1", "K2": "v2"},
		},
		{
			name: "entry with extra attributes",
			input: `[/1]
label = multiai
service = svc
key = MY_KEY
value = my-val
application = test
`,
			want: map[string]string{"MY_KEY": "my-val"},
		},
		{
			name:  "empty output",
			input: "",
			want:  map[string]string{},
		},
		{
			name: "no matching entries",
			input: `[/1]
label = other
service = other-svc
key = K
value = v
`,
			want: map[string]string{"K": "v"},
		},
		{
			name:  "only whitespace",
			input: "\n\n  \n",
			want:  map[string]string{},
		},
		{
			name: "entries without value attribute",
			input: `[/1]
label = multiai
service = svc
key = K1
value = v1

[/2]
label = multiai
service = svc
key = K2
`,
			want: map[string]string{"K1": "v1", "K2": ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseSecretToolSearch(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("len = %d, want %d\n  got:  %v\n  want: %v", len(got), len(tt.want), got, tt.want)
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("[%s] = %q, want %q", k, got[k], v)
				}
			}
		})
	}
}

// ── Integration test: end-to-end via newPlatformStore (skipped without D-Bus) ─

func TestLibsecretStore_RoundTrip(t *testing.T) {
	if _, err := exec.LookPath("secret-tool"); err != nil {
		t.Skip("secret-tool not found in PATH — skipping integration test")
	}

	store, err := newPlatformStore()
	if err != nil {
		t.Fatal(err)
	}

	service := "multiai:test-roundtrip-" + t.Name()
	key := "TEST_KEY"
	value := "test-value-roundtrip"

	// Set
	if err := store.Set(service, key, value); err != nil {
		t.Fatal(err)
	}

	// Get
	got, err := store.Get(service, key)
	if err != nil {
		t.Fatal(err)
	}
	if got != value {
		t.Errorf("roundtrip: got %q, want %q", got, value)
	}

	// List
	creds, err := store.List(service)
	if err != nil {
		t.Fatal(err)
	}
	if creds[key] != value {
		t.Errorf("List[%q] = %q, want %q", key, creds[key], value)
	}

	// Delete
	if err := store.Delete(service, key); err != nil {
		t.Fatal(err)
	}

	// Verify deleted
	_, err = store.Get(service, key)
	if err == nil {
		t.Error("expected error after delete")
	}
}
