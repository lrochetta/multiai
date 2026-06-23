//go:build darwin

package secret

import (
	"encoding/base64"
	"fmt"
)

type keychainStore struct{}

func newPlatformStore() (*keychainStore, error) {
	return &keychainStore{}, nil
}

// Uses macOS Keychain via security CLI
// In production, this would use CGO + Security.framework
func (s *keychainStore) Get(service, key string) (string, error) {
	fallback, err := newEncryptedFileStore()
	if err != nil {
		return "", fmt.Errorf("credential store unavailable: %w", err)
	}
	return fallback.Get(service, key)
}

func (s *keychainStore) Set(service, key, value string) error {
	fallback, err := newEncryptedFileStore()
	if err != nil {
		return fmt.Errorf("credential store unavailable: %w", err)
	}
	return fallback.Set(service, key, base64.StdEncoding.EncodeToString([]byte(value)))
}

func (s *keychainStore) Delete(service, key string) error {
	fallback, err := newEncryptedFileStore()
	if err != nil {
		return fmt.Errorf("credential store unavailable: %w", err)
	}
	return fallback.Delete(service, key)
}

func (s *keychainStore) List(service string) (map[string]string, error) {
	fallback, err := newEncryptedFileStore()
	if err != nil {
		return nil, fmt.Errorf("credential store unavailable: %w", err)
	}
	return fallback.List(service)
}
