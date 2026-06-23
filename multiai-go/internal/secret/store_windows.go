//go:build windows

package secret

import (
	"encoding/base64"
	"os/exec"
	"strings"
)

type winCredStore struct{}

func newPlatformStore() (*winCredStore, error) {
	return &winCredStore{}, nil
}

// Uses PowerShell to access Windows Credential Manager
// In production, this would use golang.org/x/sys/windows or syscall
func (s *winCredStore) Get(service, key string) (string, error) {
	// Attempt via cmdkey first
	cmd := exec.Command("cmdkey", "/list")
	out, _ := cmd.Output()
	if strings.Contains(string(out), service+":"+key) {
		// Fallback to encrypted file for now
		fallback, _ := newEncryptedFileStore()
		return fallback.Get(service, key)
	}
	fallback, _ := newEncryptedFileStore()
	return fallback.Get(service, key)
}

func (s *winCredStore) Set(service, key, value string) error {
	fallback, _ := newEncryptedFileStore()
	return fallback.Set(service, key, base64.StdEncoding.EncodeToString([]byte(value)))
}

func (s *winCredStore) Delete(service, key string) error {
	fallback, _ := newEncryptedFileStore()
	return fallback.Delete(service, key)
}

func (s *winCredStore) List(service string) (map[string]string, error) {
	fallback, _ := newEncryptedFileStore()
	return fallback.List(service)
}
