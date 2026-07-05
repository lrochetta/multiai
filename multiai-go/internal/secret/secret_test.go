package secret

import (
	"os"
	"strings"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	plaintext := []byte("sk-ant-api-03-test-secret-key-123456")
	ciphertext, err := encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	// Should be different from plaintext
	if string(ciphertext) == string(plaintext) {
		t.Error("ciphertext equals plaintext")
	}

	decrypted, err := decrypt(key, ciphertext)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("decrypt mismatch: got %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptDecrypt_EmptySlice(t *testing.T) {
	key := make([]byte, 32)
	plaintext := []byte("")
	ciphertext, err := encrypt(key, plaintext)
	if err != nil {
		t.Fatal(err)
	}
	decrypted, err := decrypt(key, ciphertext)
	if err != nil {
		t.Fatal(err)
	}
	if len(decrypted) != 0 {
		t.Errorf("expected empty, got %d bytes", len(decrypted))
	}
}

func TestDeriveKey(t *testing.T) {
	salt := []byte("test-salt-123456")
	key1 := DeriveKey("my-passphrase", salt)
	key2 := DeriveKey("my-passphrase", salt)
	key3 := DeriveKey("different", salt)

	if len(key1) != 32 {
		t.Errorf("key length: got %d, want 32", len(key1))
	}
	// Same inputs = same key
	for i := range key1 {
		if key1[i] != key2[i] {
			t.Error("same inputs produced different keys")
			break
		}
	}
	// Different inputs = different key
	same := true
	for i := range key1 {
		if key1[i] != key3[i] {
			same = false
			break
		}
	}
	if same {
		t.Error("different inputs produced same key")
	}
}

func TestEncryptedFileStore(t *testing.T) {
	t.Setenv("MULTIAI_SECRETS_DIR", t.TempDir())

	store, err := newEncryptedFileStore()
	if err != nil {
		t.Fatal(err)
	}

	// Set
	if err := store.Set("test-service", "API_KEY", "sk-test-abc123"); err != nil {
		t.Fatal(err)
	}

	// Get
	val, err := store.Get("test-service", "API_KEY")
	if err != nil {
		t.Fatal(err)
	}
	if val != "sk-test-abc123" {
		t.Errorf("got %q, want %q", val, "sk-test-abc123")
	}

	// List
	creds, err := store.List("test-service")
	if err != nil {
		t.Fatal(err)
	}
	if creds["API_KEY"] != "sk-test-abc123" {
		t.Errorf("list mismatch")
	}

	// Delete
	if err := store.Delete("test-service", "API_KEY"); err != nil {
		t.Fatal(err)
	}
	_, err = store.Get("test-service", "API_KEY")
	if err == nil {
		t.Error("expected error after delete")
	}
}

// TestPlatformStoreRoundTrip exercises the full public path used by
// 'multiai config' (Set) and launch-time resolution (Get) — the exact
// round-trip that was broken (base64-encoded Set, raw Get).
func TestPlatformStoreRoundTrip(t *testing.T) {
	t.Setenv("MULTIAI_SECRETS_DIR", t.TempDir())

	store, err := NewStore()
	if err != nil {
		t.Fatal(err)
	}

	service := ServiceForProfile("/some/dir/ca.env")
	if service != "multiai:ca" {
		t.Fatalf("ServiceForProfile: got %q, want %q", service, "multiai:ca")
	}

	const key = "ANTHROPIC_API_KEY"
	const value = "sk-ant-api03-roundtrip-test-1234567890"
	if err := store.Set(service, key, value); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(service, key)
	if err != nil {
		t.Fatal(err)
	}
	if got != value {
		t.Errorf("round-trip corrupted the value: got %q, want %q", got, value)
	}
}

// TestServiceNameIsFileSafe ensures a service containing ':' lands in a
// regular file inside the secrets dir (on NTFS a raw ':' would create an
// alternate data stream and no visible file).
func TestServiceNameIsFileSafe(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MULTIAI_SECRETS_DIR", dir)

	store, err := newEncryptedFileStore()
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Set("multiai:ca", "K", "v"); err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range entries {
		if e.Name() == "multiai_ca.enc" {
			found = true
		}
		if strings.Contains(e.Name(), ":") {
			t.Errorf("unsafe file name created: %q", e.Name())
		}
	}
	if !found {
		t.Errorf("expected multiai_ca.enc in %s, entries: %v", dir, entries)
	}
}
