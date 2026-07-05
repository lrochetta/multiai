//go:build windows

package secret

// winCredStore delegates to the AES-256-GCM encrypted file store.
// Native Windows Credential Manager support is planned (roadmap item 1.10);
// until then we use one honest, working backend instead of a half-stub.
type winCredStore struct {
	fallback *encryptedFileStore
}

func newPlatformStore() (*winCredStore, error) {
	fallback, err := newEncryptedFileStore()
	if err != nil {
		return nil, err
	}
	return &winCredStore{fallback: fallback}, nil
}

func (s *winCredStore) Get(service, key string) (string, error) {
	return s.fallback.Get(service, key)
}

func (s *winCredStore) Set(service, key, value string) error {
	return s.fallback.Set(service, key, value)
}

func (s *winCredStore) Delete(service, key string) error {
	return s.fallback.Delete(service, key)
}

func (s *winCredStore) List(service string) (map[string]string, error) {
	return s.fallback.List(service)
}
