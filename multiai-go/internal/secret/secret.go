// Package secret provides credential storage backed by an AES-256-GCM
// encrypted file (~/.config/multiai/secrets, override: MULTIAI_SECRETS_DIR).
//
// Threat model (be honest about it): the primary value is keeping real API
// keys OUT of the profile .env files, which are meant to be shared/committed —
// a stored secret is replaced there by the Sentinel. The at-rest encryption
// itself is only as strong as the filesystem permissions: the AES master key
// is a random blob living beside the ciphertext under the same 0600/owner, so
// anyone who can read the secrets directory (a backup, a synced folder, a
// stolen disk, another local user) can decrypt. There is no passphrase and no
// machine binding today. Native OS backends (Windows Credential Manager,
// macOS Keychain, libsecret) that would close that gap are planned —
// roadmap item 1.10 — but not implemented yet.
package secret

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/lrochetta/multiai/internal/fsutil"
)

// Sentinel is written into profile .env files in place of the real secret
// once the value has been saved in the credential store by 'multiai config'.
// At launch time the sentinel is resolved back via Store.Get.
const Sentinel = "__MULTIAI_CREDSTORE__"

// ServiceForProfile returns the credential-store service name for a profile
// .env file path. It must stay identical between the write side
// (config wizard) and the read side (launch), whatever the CWD.
func ServiceForProfile(path string) string {
	return "multiai:" + strings.TrimSuffix(filepath.Base(path), ".env")
}

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
	mu        sync.Mutex
	dir       string
	masterKey []byte
}

// secretsDir resolves the directory holding encrypted credential files.
// MULTIAI_SECRETS_DIR overrides it (tests, portable installs). os.Getenv("HOME")
// is NOT used as primary source: it is usually unset on Windows outside of
// git-bash, which silently redirected secrets into the current directory.
func secretsDir() (string, error) {
	if dir := os.Getenv("MULTIAI_SECRETS_DIR"); dir != "" {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		home = os.Getenv("HOME")
	}
	if home == "" {
		// Never fall back to a CWD-relative path: the store (masterkey
		// included) would land in whatever directory the user is in.
		return "", fmt.Errorf("cannot resolve the home directory for the credential store (set MULTIAI_SECRETS_DIR to override)")
	}
	return filepath.Join(home, ".config", "multiai", "secrets"), nil
}

func newEncryptedFileStore() (*encryptedFileStore, error) {
	dir, err := secretsDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("cannot create secrets dir: %w", err)
	}

	masterKey, err := loadOrCreateMasterKey(filepath.Join(dir, ".masterkey"))
	if err != nil {
		return nil, err
	}
	return &encryptedFileStore{dir: dir, masterKey: masterKey}, nil
}

// loadOrCreateMasterKey reads the 32-byte AES master key, or creates it once.
//
// A present-but-invalid (short/truncated) key is a HARD ERROR, never silently
// regenerated: overwriting it would orphan every existing .enc sealed under
// the old key. Creation uses O_CREATE|O_EXCL so two first-run processes racing
// on a fresh install converge on a single key (the loser reads the winner's)
// instead of each generating a different one and losing the other's ciphertext.
func loadOrCreateMasterKey(keyPath string) ([]byte, error) {
	if data, err := os.ReadFile(keyPath); err == nil {
		if len(data) < 32 {
			return nil, fmt.Errorf("master key %s is present but invalid (%d bytes < 32); refusing to overwrite it — restore it, or delete the secrets directory to reset all credentials", keyPath, len(data))
		}
		return data[:32], nil
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("cannot read master key: %w", err)
	}

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("cannot generate master key: %w", err)
	}
	f, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		if os.IsExist(err) {
			// Lost the creation race: adopt the key the winner just wrote.
			data, rerr := os.ReadFile(keyPath)
			if rerr != nil {
				return nil, fmt.Errorf("cannot read master key after creation race: %w", rerr)
			}
			if len(data) < 32 {
				return nil, fmt.Errorf("master key %s is present but invalid (%d bytes < 32)", keyPath, len(data))
			}
			return data[:32], nil
		}
		return nil, fmt.Errorf("cannot create master key: %w", err)
	}
	defer f.Close()
	if _, err := f.Write(key); err != nil {
		return nil, fmt.Errorf("cannot write master key: %w", err)
	}
	return key, nil
}

func (s *encryptedFileStore) filePath(service string) string {
	return filepath.Join(s.dir, sanitizeFileName(service)+".enc")
}

// sanitizeFileName makes a service name safe as a file name on every
// platform — a ':' (as in "multiai:ca") would otherwise create an NTFS
// alternate data stream on Windows instead of a regular file.
func sanitizeFileName(name string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9',
			r == '.', r == '-', r == '_':
			return r
		default:
			return '_'
		}
	}, name)
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
	// Zeroize plaintext after use
	defer func() {
		for i := range plaintext {
			plaintext[i] = 0
		}
	}()
	var creds map[string]string
	if err := json.Unmarshal(plaintext, &creds); err != nil {
		return nil, fmt.Errorf("credential file %s is corrupted: %w", service, err)
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
	// Atomic write: a whole .enc wraps every credential for the service under
	// one GCM blob, so a non-atomic truncate+write that is interrupted would
	// lose ALL of them. WriteFileAtomic guarantees old-or-new, never partial.
	return fsutil.WriteFileAtomic(s.filePath(service), encrypted, 0600)
}

func (s *encryptedFileStore) Get(service, key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
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
	s.mu.Lock()
	defer s.mu.Unlock()
	creds, err := s.load(service)
	if err != nil {
		return err
	}
	creds[key] = value
	return s.save(service, creds)
}

func (s *encryptedFileStore) Delete(service, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	creds, err := s.load(service)
	if err != nil {
		return err
	}
	delete(creds, key)
	return s.save(service, creds)
}

func (s *encryptedFileStore) List(service string) (map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.load(service)
}
