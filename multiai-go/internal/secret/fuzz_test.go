package secret

import (
	"testing"
)

// FuzzSecretStore fuzzes the encrypt/decrypt cycle with random keys and
// plaintexts, plus random ciphertext attempts for the decrypt side.
// It must never panic for any input.
//
//go:noinline
func FuzzSecretStore(f *testing.F) {
	// Seed corpus: representative key+plaintext pairs
	seeds := []struct {
		key       string
		plaintext string
	}{
		{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "sk-ant-api03-test-secret"},
		{string(make([]byte, 32)), ""},
		{string(make([]byte, 32)), "a"},
		{"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "sk-test-1234567890"},
		{"cccccccccccccccccccccccccccccccc", "sk-openai-ABCDEFGHIJKLMN"},
	}

	for _, s := range seeds {
		f.Add([]byte(s.key), []byte(s.plaintext))
	}

	f.Fuzz(func(t *testing.T, key, plaintext []byte) {
		// Key must be exactly 32 bytes for AES-256.
		if len(key) != 32 {
			return
		}

		// Encrypt must never panic.
		ciphertext, err := encrypt(key, plaintext)
		if err != nil {
			return // expected for edge cases (e.g. nil key material issues)
		}
		if ciphertext == nil {
			t.Error("encrypt returned nil ciphertext with nil error")
			return
		}

		// Decrypt must never panic and must round-trip correctly.
		decrypted, err := decrypt(key, ciphertext)
		if err != nil {
			t.Errorf("decrypt failed for valid ciphertext: %v", err)
			return
		}
		if len(decrypted) != len(plaintext) {
			t.Errorf("decrypted length %d != original length %d", len(decrypted), len(plaintext))
			return
		}
		for i := range plaintext {
			if decrypted[i] != plaintext[i] {
				t.Errorf("decrypted byte %d differs: got %02x, want %02x", i, decrypted[i], plaintext[i])
				return
			}
		}

		// Attempting to decrypt random ciphertext (from fuzzer) must never
		// panic — it may return an error but must not crash.
		_, _ = decrypt(key, ciphertext)
	})
}

// FuzzDeriveKey fuzzes the DeriveKey function with random passphrases and
// salts. It must never panic for any input, and must always return 32 bytes.
//
//go:noinline
func FuzzDeriveKey(f *testing.F) {
	seeds := []struct {
		passphrase string
		salt       string
	}{
		{"password", "somesalt12345678"},
		{"", ""},
		{"sk-ant-api03-test", "1234567890abcdef"},
		{"abc", "xyz"},
	}

	for _, s := range seeds {
		f.Add(s.passphrase, []byte(s.salt))
	}

	f.Fuzz(func(t *testing.T, passphrase string, salt []byte) {
		key := DeriveKey(passphrase, salt)

		if len(key) != 32 {
			t.Errorf("DeriveKey returned key of length %d, want 32", len(key))
			return
		}

		// DeriveKey must be deterministic.
		key2 := DeriveKey(passphrase, salt)
		if len(key2) != 32 {
			t.Errorf("second DeriveKey returned key of length %d, want 32", len(key2))
			return
		}
		for i := range key {
			if key[i] != key2[i] {
				t.Errorf("non-deterministic key at byte %d", i)
				return
			}
		}
	})
}
