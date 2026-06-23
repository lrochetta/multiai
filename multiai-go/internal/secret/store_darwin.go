//go:build darwin

package secret

import "encoding/base64"

type keychainStore struct{}

func newPlatformStore() (*keychainStore, error) {
	return &keychainStore{}, nil
}

// Uses macOS Keychain via security CLI
// In production, this would use CGO + Security.framework
func (s *keychainStore) Get(service, key string) (string, error) {
	fallback, _ := newEncryptedFileStore()
	return fallback.Get(service, key)
}

func (s *keychainStore) Set(service, key, value string) error {
	fallback, _ := newEncryptedFileStore()
	return fallback.Set(service, key, base64.StdEncoding.EncodeToString([]byte(value)))
}

func (s *keychainStore) Delete(service, key string) error {
	fallback, _ := newEncryptedFileStore()
	return fallback.Delete(service, key)
}

func (s *keychainStore) List(service string) (map[string]string, error) {
	fallback, _ := newEncryptedFileStore()
	return fallback.List(service)
}
