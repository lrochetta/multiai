package secret

import (
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
	t.Setenv("HOME", t.TempDir())

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
