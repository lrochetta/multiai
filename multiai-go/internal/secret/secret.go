// Package secret provides cross-platform secure credential storage.
// On Windows: Windows Credential Manager (via wincred)
// On macOS: macOS Keychain (via keychain)
// On Linux: libsecret (via freedesktop secret-service) with fallback to encrypted file
package secret

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Store abstracts secure credential storage per platform.
type Store interface {
	Get(service, key string) (string, error)
	Set(service, key, value string) error
	Delete(service, key string) error
	List(service string) (map[string]string, error)
}

// Credential represents a stored credential.
type Credential struct {
	Service string `json:"service"`
	Key     string `json:"key"`
	Value   string `json:"value"`
}

// NewStore returns the best available credential store for the current platform.
func NewStore() (Store, error) {
	return newPlatformStore()
}

// ── Encrypted File Store (fallback pour Linux) ──────────────────────────────

type encryptedFileStore struct {
	dir       string
	masterKey []byte
}

func newEncryptedFileStore() (*encryptedFileStore, error) {
	dir := filepath.Join(os.Getenv("HOME"), ".config", "multiai", "secrets")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("cannot create secrets dir: %w", err)
	}

	// Derive master key from machine-id or random seed
	keyPath := filepath.Join(dir, ".masterkey")
	var masterKey []byte
	if data, err := os.ReadFile(keyPath); err == nil && len(data) >= 32 {
		masterKey = data[:32]
	} else {
		masterKey = make([]byte, 32)
		if _, err := rand.Read(masterKey); err != nil {
			return nil, fmt.Errorf("cannot generate master key: %w", err)
		}
		if err := os.WriteFile(keyPath, masterKey, 0600); err != nil {
			return nil, fmt.Errorf("cannot write master key: %w", err)
		}
	}

	return &encryptedFileStore{dir: dir, masterKey: masterKey}, nil
}

func (s *encryptedFileStore) filePath(service string) string {
	return filepath.Join(s.dir, service+".enc")
}

func (s *encryptedFileStore) load(service string) (map[string]string, error) {
	data, err := os.ReadFile(s.filePath(service))
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]string), nil
		}
		return nil, err
	}
	plaintext, err := decrypt(s.masterKey, data)
	if err != nil {
		return nil, err
	}
	var creds map[string]string
	if err := json.Unmarshal(plaintext, &creds); err != nil {
		return make(map[string]string), nil
	}
	return creds, nil
}

func (s *encryptedFileStore) save(service string, creds map[string]string) error {
	plaintext, err := json.Marshal(creds)
	if err != nil {
		return err
	}
	encrypted, err := encrypt(s.masterKey, plaintext)
	if err != nil {
		return err
	}
	return os.WriteFile(s.filePath(service), encrypted, 0600)
}

func (s *encryptedFileStore) Get(service, key string) (string, error) {
	creds, err := s.load(service)
	if err != nil {
		return "", err
	}
	if v, ok := creds[key]; ok {
		return v, nil
	}
	return "", fmt.Errorf("credential not found: %s/%s", service, key)
}

func (s *encryptedFileStore) Set(service, key, value string) error {
	creds, err := s.load(service)
	if err != nil {
		return err
	}
	creds[key] = value
	return s.save(service, creds)
}

func (s *encryptedFileStore) Delete(service, key string) error {
	creds, err := s.load(service)
	if err != nil {
		return err
	}
	delete(creds, key)
	return s.save(service, creds)
}

func (s *encryptedFileStore) List(service string) (map[string]string, error) {
	return s.load(service)
}
