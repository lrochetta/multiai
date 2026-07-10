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
	"crypto/sha256"
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
// .env file path. It includes a SHA-256 hash of the canonical absolute path
// to prevent name collision when two profiles in different directories share
// the same file name (see audit finding S2.4).
//
// The format is "multiai:<basename>-<first8hexbytes>", e.g.
// "multiai:ca-a1b2c3d4e5f6a7b8".
//
// It must stay identical between the write side (config wizard) and the
// read side (launch), whatever the CWD.
func ServiceForProfile(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	canonical := filepath.Clean(abs)
	h := sha256.Sum256([]byte(canonical))
	hashSuffix := fmt.Sprintf("%x", h[:8])
	base := strings.TrimSuffix(filepath.Base(canonical), filepath.Ext(canonical))
	return fmt.Sprintf("multiai:%s-%s", base, hashSuffix)
}

// oldServiceForProfile returns the previous credential-store service name
// (basename-only, no hash). Used by migrateServiceName to look up secrets
// stored before the namespace fix (S2.4).
func oldServiceForProfile(path string) string {
	return "multiai:" + strings.TrimSuffix(filepath.Base(path), ".env")
}

// MigrateServiceName migrates credentials from the old service name
// (basename-only) to the new service name (canonical-path-hashed) when
// they were stored before the namespace fix. It reads data from the old
// service, writes it to the new service, and cleans up the old entry.
//
// Callers MUST pass the new-format service name for normal operations;
// this function computes the old name internally from the profile path.
func MigrateServiceName(store Store, profilePath string) (string, error) {
	newService := ServiceForProfile(profilePath)
	oldService := oldServiceForProfile(profilePath)

	// Same name means nothing changed — no migration needed.
	if newService == oldService {
		return newService, nil
	}

	// Read old credentials, if any.
	oldCreds, err := store.List(oldService)
	if err != nil {
		// Old service doesn't exist or can't be read — nothing to migrate.
		return newService, nil
	}
	if len(oldCreds) == 0 {
		return newService, nil
	}

	// Only migrate if the new service doesn't already have data
	// (avoid overwriting/duplicating after a prior partial migration).
	newCreds, err := store.List(newService)
	if err != nil || len(newCreds) > 0 {
		return newService, nil
	}

	// Migrate each key, then delete from old service.
	for k, v := range oldCreds {
		if err := store.Set(newService, k, v); err != nil {
			return newService, fmt.Errorf("migration: set %s/%s: %w", newService, k, err)
		}
	}
	for k := range oldCreds {
		if err := store.Delete(oldService, k); err != nil {
			return newService, fmt.Errorf("migration: delete %s/%s: %w", oldService, k, err)
		}
	}

	return newService, nil
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
	mu      sync.Mutex
	dir     string
	keyPath string // never holds the key material — only the filesystem path
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

	keyPath := filepath.Join(dir, ".masterkey")
	// Ensure the master key file exists (create once, race-safe). The key is
	// loaded and zeroised per-operation (loadMasterKey) so it does NOT live in
	// this struct between calls — see S5.5 zeroisation.
	masterKey, err := loadOrCreateMasterKey(keyPath)
	if err != nil {
		return nil, err
	}
	Zeroize(masterKey)
	return &encryptedFileStore{dir: dir, keyPath: keyPath}, nil
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
		Zeroize(key)
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
		Zeroize(key)
		return nil, fmt.Errorf("cannot write master key: %w", err)
	}
	return key, nil
}

func (s *encryptedFileStore) filePath(service string) string {
	return filepath.Join(s.dir, sanitizeFileName(service)+".enc")
}

// lockPath returns the path to the inter-process lock file for the given
// service. A dedicated .lock file (never replaced) avoids the problem of
// locking the .enc file itself, which gets atomically renamed during save —
// a handle on the old inode would not block a handle on the new inode.
func (s *encryptedFileStore) lockPath(service string) string {
	return filepath.Join(s.dir, sanitizeFileName(service)+".lock")
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

// loadMasterKey reads the 32-byte AES master key from disk.
// The caller MUST zeroise the returned slice via defer Zeroize(key).
func (s *encryptedFileStore) loadMasterKey() ([]byte, error) {
	data, err := os.ReadFile(s.keyPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read master key: %w", err)
	}
	if len(data) < 32 {
		return nil, fmt.Errorf("master key %s is invalid (%d bytes < 32)", s.keyPath, len(data))
	}
	return data[:32], nil
}

func (s *encryptedFileStore) load(service string, key []byte) (map[string]string, error) {
	data, err := os.ReadFile(s.filePath(service))
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]string), nil
		}
		return nil, err
	}
	plaintext, err := decrypt(key, data)
	if err != nil {
		return nil, err
	}
	// Zeroize plaintext after use — the caller gets a map of strings (immutable),
	// the decrypted bytes must not linger in the heap.
	defer Zeroize(plaintext)
	var creds map[string]string
	if err := json.Unmarshal(plaintext, &creds); err != nil {
		return nil, fmt.Errorf("credential file %s is corrupted: %w", service, err)
	}
	return creds, nil
}

func (s *encryptedFileStore) save(service string, creds map[string]string, key []byte) error {
	plaintext, err := json.Marshal(creds)
	if err != nil {
		return err
	}
	defer Zeroize(plaintext)
	encrypted, err := encrypt(key, plaintext)
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

	masterKey, err := s.loadMasterKey()
	if err != nil {
		return "", err
	}
	defer Zeroize(masterKey)

	creds, err := s.load(service, masterKey)
	if err != nil {
		return "", err
	}
	if v, ok := creds[key]; ok {
		return v, nil
	}
	return "", fmt.Errorf("credential not found: %s/%s", service, key)
}

func (s *encryptedFileStore) Set(service, key, value string) error {
	release, err := s.lockService(service)
	if err != nil {
		return err
	}
	defer release()

	s.mu.Lock()
	defer s.mu.Unlock()

	masterKey, err := s.loadMasterKey()
	if err != nil {
		return err
	}
	defer Zeroize(masterKey)

	creds, err := s.load(service, masterKey)
	if err != nil {
		return err
	}
	creds[key] = value
	return s.save(service, creds, masterKey)
}

func (s *encryptedFileStore) Delete(service, key string) error {
	release, err := s.lockService(service)
	if err != nil {
		return err
	}
	defer release()

	s.mu.Lock()
	defer s.mu.Unlock()

	masterKey, err := s.loadMasterKey()
	if err != nil {
		return err
	}
	defer Zeroize(masterKey)

	creds, err := s.load(service, masterKey)
	if err != nil {
		return err
	}
	delete(creds, key)
	return s.save(service, creds, masterKey)
}

func (s *encryptedFileStore) List(service string) (map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	masterKey, err := s.loadMasterKey()
	if err != nil {
		return nil, err
	}
	defer Zeroize(masterKey)

	return s.load(service, masterKey)
}
