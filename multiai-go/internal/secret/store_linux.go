//go:build linux

package secret

type libsecretStore struct {
	fallback *encryptedFileStore
}

func newPlatformStore() (Store, error) {
	// Attempt D-Bus connection to secret-service
	// If unavailable, fallback to encrypted file
	fallback, err := newEncryptedFileStore()
	if err != nil {
		return nil, err
	}
	return &libsecretStore{fallback: fallback}, nil
}

func (s *libsecretStore) Get(service, key string) (string, error) {
	return s.fallback.Get(service, key)
}

func (s *libsecretStore) Set(service, key, value string) error {
	return s.fallback.Set(service, key, value)
}

func (s *libsecretStore) Delete(service, key string) error {
	return s.fallback.Delete(service, key)
}

func (s *libsecretStore) List(service string) (map[string]string, error) {
	return s.fallback.List(service)
}
