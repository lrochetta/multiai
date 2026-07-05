//go:build darwin

package secret

// keychainStore delegates to the AES-256-GCM encrypted file store.
// Native macOS Keychain support is planned (roadmap item 1.10);
// until then we use one honest, working backend instead of a half-stub.
type keychainStore struct {
	fallback *encryptedFileStore
}

func newPlatformStore() (*keychainStore, error) {
	fallback, err := newEncryptedFileStore()
	if err != nil {
		return nil, err
	}
	return &keychainStore{fallback: fallback}, nil
}

func (s *keychainStore) Get(service, key string) (string, error) {
	return s.fallback.Get(service, key)
}

func (s *keychainStore) Set(service, key, value string) error {
	return s.fallback.Set(service, key, value)
}

func (s *keychainStore) Delete(service, key string) error {
	return s.fallback.Delete(service, key)
}

func (s *keychainStore) List(service string) (map[string]string, error) {
	return s.fallback.List(service)
}
