package secret

import (
	"testing"
)

// ── extractQuotedValue tests ─────────────────────────────────────────────

func TestExtractQuotedValue_Simple(t *testing.T) {
	line := `0x00000007 <blob> = "account-name"`
	got := extractQuotedValue(line)
	if got != "account-name" {
		t.Errorf("extractQuotedValue(%q) = %q, want %q", line, got, "account-name")
	}
}

func TestExtractQuotedValue_EmptyQuotes(t *testing.T) {
	line := `0x00000007 <blob> = ""`
	got := extractQuotedValue(line)
	if got != "" {
		t.Errorf("extractQuotedValue(%q) = %q, want ''", line, got)
	}
}

func TestExtractQuotedValue_NoQuotes(t *testing.T) {
	line := `0x00000007 <blob> = nothing`
	got := extractQuotedValue(line)
	if got != "" {
		t.Errorf("extractQuotedValue(%q) = %q, want ''", line, got)
	}
}

func TestExtractQuotedValue_EmptyString(t *testing.T) {
	if got := extractQuotedValue(""); got != "" {
		t.Errorf("extractQuotedValue('') = %q, want ''", got)
	}
}

func TestExtractQuotedValue_SingleQuote(t *testing.T) {
	line := `0x00000007 <blob> = "unclosed`
	got := extractQuotedValue(line)
	if got != "" {
		t.Errorf("extractQuotedValue(%q) = %q, want '' (unclosed quote)", line, got)
	}
}

func TestExtractQuotedValue_SpecialChars(t *testing.T) {
	line := `0x00000007 <blob> = "hello-world_123.foo"`
	got := extractQuotedValue(line)
	if got != "hello-world_123.foo" {
		t.Errorf("extractQuotedValue(%q) = %q, want %q", line, got, "hello-world_123.foo")
	}
}

func TestExtractQuotedValue_QuotedSpaces(t *testing.T) {
	line := `0x00000007 <blob> = "My Keychain Item"`
	got := extractQuotedValue(line)
	if got != "My Keychain Item" {
		t.Errorf("extractQuotedValue(%q) = %q, want %q", line, got, "My Keychain Item")
	}
}

// ── parseDumpKeychain tests ──────────────────────────────────────────────

func TestParseDumpKeychain_SingleEntry(t *testing.T) {
	input := `keychain: "/path/to/login.keychain-db"
    version: 512
    class: "genp"
    attributes:
        0x00000007 <blob> = "my-account"
        0x00000008 <blob> = "my-service"
`
	entries := parseDumpKeychain(input)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Account != "my-account" {
		t.Errorf("Account = %q, want %q", entries[0].Account, "my-account")
	}
	if entries[0].Service != "my-service" {
		t.Errorf("Service = %q, want %q", entries[0].Service, "my-service")
	}
}

func TestParseDumpKeychain_MultipleEntries(t *testing.T) {
	input := `keychain: "/path/to/login.keychain-db"
    version: 512
    class: "genp"
    attributes:
        0x00000007 <blob> = "account-1"
        0x00000008 <blob> = "service-1"

keychain: "/path/to/login.keychain-db"
    version: 512
    class: "genp"
    attributes:
        0x00000007 <blob> = "account-2"
        0x00000008 <blob> = "service-2"
`
	entries := parseDumpKeychain(input)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Account != "account-1" || entries[0].Service != "service-1" {
		t.Errorf("entry 0: got (%q, %q), want (account-1, service-1)", entries[0].Account, entries[0].Service)
	}
	if entries[1].Account != "account-2" || entries[1].Service != "service-2" {
		t.Errorf("entry 1: got (%q, %q), want (account-2, service-2)", entries[1].Account, entries[1].Service)
	}
}

func TestParseDumpKeychain_FiltersNonGenp(t *testing.T) {
	// A block with class "inet" (not genp) should be skipped.
	input := `keychain: "/path/to/login.keychain-db"
    version: 512
    class: "inet"
    attributes:
        0x00000007 <blob> = "my-account"
        0x00000008 <blob> = "my-service"

keychain: "/path/to/login.keychain-db"
    version: 512
    class: "genp"
    attributes:
        0x00000007 <blob> = "real-account"
        0x00000008 <blob> = "real-service"
`
	entries := parseDumpKeychain(input)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry (inet filtered), got %d", len(entries))
	}
	if entries[0].Account != "real-account" {
		t.Errorf("Account = %q, want %q", entries[0].Account, "real-account")
	}
}

func TestParseDumpKeychain_EmptyInput(t *testing.T) {
	entries := parseDumpKeychain("")
	if entries != nil {
		t.Errorf("expected nil for empty input, got %v", entries)
	}
}

func TestParseDumpKeychain_OnlyWhitespace(t *testing.T) {
	entries := parseDumpKeychain("\n\n  \n")
	if len(entries) != 0 {
		t.Errorf("expected 0 entries for whitespace-only input, got %d", len(entries))
	}
}

func TestParseDumpKeychain_MissingAccountOrService(t *testing.T) {
	// Entry with service but no account should be skipped.
	input := `keychain: "/path/to/login.keychain-db"
    version: 512
    class: "genp"
    attributes:
        0x00000008 <blob> = "only-service"
`
	entries := parseDumpKeychain(input)
	if len(entries) != 0 {
		t.Errorf("expected 0 entries (missing account), got %d", len(entries))
	}
}

func TestParseDumpKeychain_WithExtraAttributes(t *testing.T) {
	// Extra attributes (other OIDs) should be ignored.
	input := `keychain: "/path/to/login.keychain-db"
    version: 512
    class: "genp"
    attributes:
        0x00000007 <blob> = "my-account"
        0x00000008 <blob> = "my-service"
        0x00000009 <blob> = "extra-value"
`
	entries := parseDumpKeychain(input)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry (extra attr ignored), got %d", len(entries))
	}
	if entries[0].Account != "my-account" || entries[0].Service != "my-service" {
		t.Errorf("entry: got (%q, %q), want (my-account, my-service)", entries[0].Account, entries[0].Service)
	}
}

func TestParseDumpKeychain_UnorderedAttributes(t *testing.T) {
	// Service (0x00000008) before account (0x00000007) should still work.
	input := `keychain: "/path/to/login.keychain-db"
    version: 512
    class: "genp"
    attributes:
        0x00000008 <blob> = "my-service"
        0x00000007 <blob> = "my-account"
`
	entries := parseDumpKeychain(input)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Account != "my-account" || entries[0].Service != "my-service" {
		t.Errorf("entry: got (%q, %q), want (my-account, my-service)", entries[0].Account, entries[0].Service)
	}
}

func TestParseDumpKeychain_NoKeychainHeader(t *testing.T) {
	// Even without a "keychain:" header, a genp block should be parsed.
	input := `version: 512
    class: "genp"
    attributes:
        0x00000007 <blob> = "orphan-account"
        0x00000008 <blob> = "orphan-service"
`
	entries := parseDumpKeychain(input)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry (no header), got %d", len(entries))
	}
}

// ── splitLines helper tests ──────────────────────────────────────────────

func TestSplitLines_Empty(t *testing.T) {
	lines := splitLines("")
	if len(lines) != 1 || lines[0] != "" {
		t.Errorf("splitLines('') = %v, want ['']", lines)
	}
}

func TestSplitLines_Multiple(t *testing.T) {
	lines := splitLines("a\nb\nc")
	if len(lines) != 3 || lines[0] != "a" || lines[1] != "b" || lines[2] != "c" {
		t.Errorf("splitLines = %v, want [a b c]", lines)
	}
}

func TestSplitLines_TrailingNewline(t *testing.T) {
	lines := splitLines("a\nb\n")
	if len(lines) != 3 || lines[2] != "" {
		t.Errorf("splitLines with trailing newline = %v, want [a b '']", lines)
	}
}

// ── joinLines helper tests ───────────────────────────────────────────────

func TestJoinLines(t *testing.T) {
	got := joinLines([]string{"a", "b", "c"})
	if got != "a\nb\nc" {
		t.Errorf("joinLines = %q, want %q", got, "a\nb\nc")
	}
}

func TestJoinLines_Single(t *testing.T) {
	got := joinLines([]string{"only"})
	if got != "only" {
		t.Errorf("joinLines single = %q, want %q", got, "only")
	}
}

func TestJoinLines_Empty(t *testing.T) {
	got := joinLines([]string{})
	if got != "" {
		t.Errorf("joinLines empty = %q, want ''", got)
	}
}

// ── trimSpace helper tests ───────────────────────────────────────────────

func TestTrimSpace_Spaces(t *testing.T) {
	if got := trimSpace("  hello  "); got != "hello" {
		t.Errorf("trimSpace = %q, want %q", got, "hello")
	}
}

func TestTrimSpace_Tabs(t *testing.T) {
	if got := trimSpace("\t\thello\t"); got != "hello" {
		t.Errorf("trimSpace = %q, want %q", got, "hello")
	}
}

func TestTrimSpace_Mixed(t *testing.T) {
	if got := trimSpace(" \t foo\t "); got != "foo" {
		t.Errorf("trimSpace = %q, want %q", got, "foo")
	}
}

func TestTrimSpace_Empty(t *testing.T) {
	if got := trimSpace(""); got != "" {
		t.Errorf("trimSpace('') = %q, want ''", got)
	}
}

func TestTrimSpace_Noop(t *testing.T) {
	if got := trimSpace("hello"); got != "hello" {
		t.Errorf("trimSpace = %q, want %q", got, "hello")
	}
}

// ── hasPrefix helper tests ───────────────────────────────────────────────

func TestHasPrefix(t *testing.T) {
	if !hasPrefix("hello world", "hello") {
		t.Error("hasPrefix should be true")
	}
	if hasPrefix("hello world", "world") {
		t.Error("hasPrefix should be false")
	}
	if hasPrefix("hi", "hello") {
		t.Error("hasPrefix should be false for longer prefix")
	}
	if hasPrefix("", "") {
		t.Error("hasPrefix('', '') should be false")
	}
}

// ── indexByte / lastIndexByte helper tests ───────────────────────────────

func TestIndexByte(t *testing.T) {
	if got := indexByte("hello", 'l'); got != 2 {
		t.Errorf("indexByte('hello', 'l') = %d, want 2", got)
	}
	if got := indexByte("hello", 'z'); got != -1 {
		t.Errorf("indexByte('hello', 'z') = %d, want -1", got)
	}
	if got := indexByte("", 'a'); got != -1 {
		t.Errorf("indexByte('', 'a') = %d, want -1", got)
	}
}

func TestLastIndexByte(t *testing.T) {
	if got := lastIndexByte("hello", 'l'); got != 3 {
		t.Errorf("lastIndexByte('hello', 'l') = %d, want 3", got)
	}
	if got := lastIndexByte("hello", 'z'); got != -1 {
		t.Errorf("lastIndexByte('hello', 'z') = %d, want -1", got)
	}
	if got := lastIndexByte("", 'a'); got != -1 {
		t.Errorf("lastIndexByte('', 'a') = %d, want -1", got)
	}
	if got := lastIndexByte("a", 'a'); got != 0 {
		t.Errorf("lastIndexByte('a', 'a') = %d, want 0", got)
	}
}

// ── isKeychainHeader / containsGenp helper tests ─────────────────────────

func TestIsKeychainHeader(t *testing.T) {
	if !isKeychainHeader(`keychain: "/path/to/keychain"`) {
		t.Error("should detect keychain header")
	}
	if !isKeychainHeader(`  keychain: "/path"`) {
		t.Error("should detect indented keychain header")
	}
	if isKeychainHeader(`version: 512`) {
		t.Error("should not detect version as keychain header")
	}
	if isKeychainHeader("") {
		t.Error("should not detect empty string as keychain header")
	}
}

func TestContainsGenp(t *testing.T) {
	if !containsGenp(`class: "genp"`) {
		t.Error("should detect genp class")
	}
	if containsGenp(`class: "inet"`) {
		t.Error("should not detect inet as genp")
	}
	if containsGenp("") {
		t.Error("should not detect genp in empty string")
	}
}

// ── indexOf helper tests ────────────────────────────────────────────────

func TestIndexOf(t *testing.T) {
	if got := indexOf("hello world", "world"); got != 6 {
		t.Errorf("indexOf = %d, want 6", got)
	}
	if got := indexOf("hello world", "xyz"); got != -1 {
		t.Errorf("indexOf = %d, want -1", got)
	}
	if got := indexOf("hello", ""); got != 0 {
		t.Errorf("indexOf('', empty) = %d, want 0", got)
	}
}

// ── Round-trip: splitLines + joinLines ──────────────────────────────────

func TestSplitJoinRoundTrip(t *testing.T) {
	input := "line1\nline2\nline3"
	lines := splitLines(input)
	rejoined := joinLines(lines)
	if rejoined != input {
		t.Errorf("split+join round-trip: got %q, want %q", rejoined, input)
	}
}
