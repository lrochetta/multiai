//go:build darwin

package secret

import "fmt"

// keychainStore stores credentials in the macOS Keychain.
//
// When CGo is available (the normal case on macOS) it uses the Security
// Framework directly via store_darwin_cgo.go.  When CGo is disabled the
// package falls back to shelling out to /usr/bin/security (store_darwin_nocgo.go).
//
// Service names are mapped to kSecAttrService, credential keys to
// kSecAttrAccount, and values to kSecValueData.
type keychainStore struct{}

func newPlatformStore() (Store, error) {
	if keychainAvailable() {
		return &keychainStore{}, nil
	}
	return newEncryptedFileStore()
}

// newNamedStore returns the requested named backend on macOS.
func newNamedStore(backend string) (Store, error) {
	switch backend {
	case "keychain":
		return newPlatformStore()
	default:
		return nil, fmt.Errorf("unsupported backend on this platform: %s (supported: keychain, file, auto)", backend)
	}
}

func (s *keychainStore) Get(service, key string) (string, error) {
	return keychainGet(service, key)
}

func (s *keychainStore) Set(service, key, value string) error {
	return keychainSet(service, key, value)
}

func (s *keychainStore) Delete(service, key string) error {
	return keychainDelete(service, key)
}

func (s *keychainStore) List(service string) (map[string]string, error) {
	return keychainList(service)
}

