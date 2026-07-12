package secret

import (
	"testing"
)

// ── targetName tests ─────────────────────────────────────────────────────

func TestTargetName_Format(t *testing.T) {
	tn := targetName("my-service", "API_KEY")
	// Format: mti_<16hex>_<key>
	if len(tn) < 22 {
		t.Fatalf("targetName too short: %q (len=%d)", tn, len(tn))
	}
	if tn[:4] != "mti_" {
		t.Errorf("targetName missing prefix: got %q, want prefix mti_", tn[:4])
	}
	if tn[20] != '_' {
		t.Errorf("targetName missing separator at position 20: got %q", tn[20])
	}
	if tn[21:] != "API_KEY" {
		t.Errorf("targetName key part: got %q, want API_KEY", tn[21:])
	}
	// Verify hex prefix is 16 hex chars
	hexPart := tn[4:20]
	for i, c := range hexPart {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("hexPart[%d] = %c, want hex digit", i, c)
		}
	}
}

func TestTargetName_DifferentService_DifferentHash(t *testing.T) {
	t1 := targetName("service-a", "KEY")
	t2 := targetName("service-b", "KEY")
	h1 := t1[4:20]
	h2 := t2[4:20]
	if h1 == h2 {
		t.Errorf("different services should produce different hash prefixes: both %q", h1)
	}
}

func TestTargetName_SameService_SameHash(t *testing.T) {
	t1 := targetName("same-service", "KEY")
	t2 := targetName("same-service", "KEY")
	if t1 != t2 {
		t.Errorf("same inputs should produce identical targetName:\n  t1=%q\n  t2=%q", t1, t2)
	}
}

func TestTargetName_SameService_DifferentKey(t *testing.T) {
	t1 := targetName("svc", "KEY_A")
	t2 := targetName("svc", "KEY_B")
	if t1[4:20] != t2[4:20] {
		t.Error("same service must produce same hash prefix")
	}
	if t1[21:] != "KEY_A" || t2[21:] != "KEY_B" {
		t.Errorf("different keys should produce different suffixes: %q vs %q", t1[21:], t2[21:])
	}
}

// ── serviceFilter tests ──────────────────────────────────────────────────

func TestServiceFilter_Format(t *testing.T) {
	sf := serviceFilter("my-service")
	// Format: mti_<16hex>_*
	if len(sf) < 22 {
		t.Fatalf("serviceFilter too short: %q (len=%d)", sf, len(sf))
	}
	if sf[:4] != "mti_" {
		t.Errorf("serviceFilter missing prefix: got %q, want prefix mti_", sf[:4])
	}
	if sf[20:] != "_*" {
		t.Errorf("serviceFilter suffix: got %q, want _*", sf[20:])
	}
}

func TestServiceFilter_ConsistentWithTargetName(t *testing.T) {
	tn := targetName("my-service", "ANY_KEY")
	sf := serviceFilter("my-service")
	// The hash prefix should be identical.
	if tn[4:20] != sf[4:20] {
		t.Errorf("hash mismatch:\n  targetName hash:  %q\n  serviceFilter hash: %q", tn[4:20], sf[4:20])
	}
}

func TestServiceFilter_DifferentService_DifferentFilter(t *testing.T) {
	sf1 := serviceFilter("service-a")
	sf2 := serviceFilter("service-b")
	if sf1[4:20] == sf2[4:20] {
		t.Error("different services should produce different filter hash prefixes")
	}
}

// ── extractKey tests ─────────────────────────────────────────────────────

func TestExtractKey_Valid(t *testing.T) {
	tn := targetName("some-service", "ANTHROPIC_API_KEY")
	key := extractKey(tn)
	if key != "ANTHROPIC_API_KEY" {
		t.Errorf("extractKey(%q) = %q, want ANTHROPIC_API_KEY", tn, key)
	}
}

func TestExtractKey_ValidWithSpecialChars(t *testing.T) {
	tn := targetName("svc", "my-key_123")
	key := extractKey(tn)
	if key != "my-key_123" {
		t.Errorf("extractKey(%q) = %q, want my-key_123", tn, key)
	}
}

func TestExtractKey_EmptyString(t *testing.T) {
	if key := extractKey(""); key != "" {
		t.Errorf("extractKey('') = %q, want ''", key)
	}
}

func TestExtractKey_TooShort(t *testing.T) {
	if key := extractKey("mti_abc"); key != "" {
		t.Errorf("extractKey(%q) = %q, want ''", "mti_abc", key)
	}
}

func TestExtractKey_WrongPrefix(t *testing.T) {
	if key := extractKey("xxx_a1b2c3d4e5f6a7b8_KEY"); key != "" {
		t.Errorf("extractKey should return '' for wrong prefix, got %q", key)
	}
}

func TestExtractKey_MissingUnderscoreSeparator(t *testing.T) {
	// Valid 16 hex chars but no trailing underscore before key.
	if key := extractKey("mti_a1b2c3d4e5f6a7b8KEY"); key != "" {
		t.Errorf("extractKey should return '' when separator is missing, got %q", key)
	}
}

func TestExtractKey_InvalidHexCharacters(t *testing.T) {
	if key := extractKey("mti_zzzzzzzzzzzzzzzz_KEY"); key != "" {
		t.Errorf("extractKey should return '' for invalid hex chars, got %q", key)
	}
}

func TestExtractKey_KeyWithUnderscore(t *testing.T) {
	tn := targetName("svc", "MY_KEY_WITH_UNDERSCORES")
	key := extractKey(tn)
	if key != "MY_KEY_WITH_UNDERSCORES" {
		t.Errorf("extractKey(%q) = %q, want MY_KEY_WITH_UNDERSCORES", tn, key)
	}
}

func TestExtractKey_EmptyKeyPart(t *testing.T) {
	// Construct a target name with an empty key part.
	tn := "mti_a1b2c3d4e5f6a7b8_"
	key := extractKey(tn)
	if key != "" {
		t.Errorf("extractKey should return '' for empty key part, got %q", key)
	}
}

func TestExtractKey_UpperCaseHex(t *testing.T) {
	// extractKey only accepts lowercase hex.
	if key := extractKey("mti_A1B2C3D4E5F6A7B8_KEY"); key != "" {
		t.Errorf("extractKey should reject uppercase hex, got %q", key)
	}
}

// ── Round-trip: targetName → extractKey ─────────────────────────────────

func TestTargetNameExtractKeyRoundTrip(t *testing.T) {
	keys := []string{
		"API_KEY",
		"a",
		"OPENAI_API_KEY",
		"very-long-key-name-that-should-still-work-fine",
		"key-with-dashes",
		"key.with.dots",
	}
	for _, k := range keys {
		tn := targetName("test-service", k)
		got := extractKey(tn)
		if got != k {
			t.Errorf("round-trip [%q]: targetName → extractKey = %q, want %q", k, got, k)
		}
	}
}

func TestExtractKeyWithServiceFilter(t *testing.T) {
	// Ensure that extractKey can parse keys from target names built with the
	// same service that was used to create the service filter.
	service := "test-service"
	sf := serviceFilter(service)
	// serviceFilter ends with _*, while targetName ends with _<key>.
	// Extract the common prefix.
	commonPrefix := sf[:20] // mti_<16hex>
	tn := targetName(service, "MY_KEY")
	if tn[:20] != commonPrefix {
		t.Errorf("targetName prefix doesn't match serviceFilter prefix:\n  tn:  %q\n  sf:  %q", tn[:20], commonPrefix)
	}
	key := extractKey(tn)
	if key != "MY_KEY" {
		t.Errorf("extractKey(%q) = %q, want MY_KEY", tn, key)
	}
}
