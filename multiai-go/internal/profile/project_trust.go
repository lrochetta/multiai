package profile

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/lrochetta/multiai/internal/fsutil"
)

const (
	projectTrustStoreVersion  = 1
	maxProjectTrustStoreSize  = 1 << 20
	maxTrustedProjectYAMLSize = 1 << 20
)

var (
	// ErrProjectConfigUntrusted means that the project configuration has never
	// been explicitly approved by the current user.
	ErrProjectConfigUntrusted = errors.New("project configuration is not trusted")
	// ErrProjectConfigChanged means that the configuration was approved before,
	// but its content no longer matches the approved SHA-256 fingerprint.
	ErrProjectConfigChanged = errors.New("trusted project configuration has changed")
	// ErrProjectTrustStoreCorrupt means that the trust store cannot safely be
	// interpreted. Callers must fail closed and must not overwrite it implicitly.
	ErrProjectTrustStoreCorrupt = errors.New("project trust store is corrupt")

	projectTrustMu sync.Mutex
)

// ProjectTrustState describes the relationship between a project config and
// the user's trust store.
type ProjectTrustState string

const (
	ProjectTrustUntrusted ProjectTrustState = "untrusted"
	ProjectTrustTrusted   ProjectTrustState = "trusted"
	ProjectTrustChanged   ProjectTrustState = "changed"
)

// ProjectTrustStatus is safe to display before asking a user for approval.
// Fingerprint is the SHA-256 of the bytes inspected in CanonicalPath.
type ProjectTrustStatus struct {
	CanonicalPath      string            `json:"canonical_path"`
	Fingerprint        string            `json:"sha256"`
	TrustedFingerprint string            `json:"trusted_sha256,omitempty"`
	TrustedAt          string            `json:"trusted_at,omitempty"`
	State              ProjectTrustState `json:"state"`
}

// Trusted reports whether the currently inspected bytes are explicitly
// approved. It deliberately does not treat a previously trusted path as safe
// when the fingerprint changed.
func (s ProjectTrustStatus) Trusted() bool {
	return s.State == ProjectTrustTrusted
}

// ProjectTrustError carries the status needed to explain a fail-closed trust
// decision while remaining compatible with errors.Is.
type ProjectTrustError struct {
	Status ProjectTrustStatus
	cause  error
}

func (e *ProjectTrustError) Error() string {
	if errors.Is(e.cause, ErrProjectConfigChanged) {
		return fmt.Sprintf("%s: %s (approved %s, current %s)", e.cause, e.Status.CanonicalPath, e.Status.TrustedFingerprint, e.Status.Fingerprint)
	}
	return fmt.Sprintf("%s: %s (sha256 %s)", e.cause, e.Status.CanonicalPath, e.Status.Fingerprint)
}

func (e *ProjectTrustError) Unwrap() error { return e.cause }

// ProjectTrustStore persists explicit approvals in the current user's config
// directory. Use NewProjectTrustStore with a temporary path in tests.
type ProjectTrustStore struct {
	path string
}

type projectTrustFile struct {
	Version  int                          `json:"version"`
	Projects map[string]projectTrustEntry `json:"projects"`
}

type projectTrustEntry struct {
	CanonicalPath string `json:"canonical_path"`
	Fingerprint   string `json:"sha256"`
	TrustedAt     string `json:"trusted_at"`
}

// NewProjectTrustStore returns a store backed by path.
func NewProjectTrustStore(path string) *ProjectTrustStore {
	return &ProjectTrustStore{path: filepath.Clean(path)}
}

// DefaultProjectTrustStore returns the user-scoped trust store. No project
// file or repository-local state can redirect this location.
func DefaultProjectTrustStore() (*ProjectTrustStore, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("locate user config directory: %w", err)
	}
	return NewProjectTrustStore(filepath.Join(configDir, "multiai", "trusted-projects.json")), nil
}

// Path returns the backing trust-store path.
func (s *ProjectTrustStore) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

// InspectProjectConfigTrust fingerprints path and reports its trust state
// without mutating the trust store.
func InspectProjectConfigTrust(path string) (ProjectTrustStatus, error) {
	store, err := DefaultProjectTrustStore()
	if err != nil {
		return ProjectTrustStatus{}, err
	}
	return store.Inspect(path)
}

// TrustProjectConfig explicitly approves the current canonical path and bytes.
func TrustProjectConfig(path string) (ProjectTrustStatus, error) {
	store, err := DefaultProjectTrustStore()
	if err != nil {
		return ProjectTrustStatus{}, err
	}
	return store.Trust(path)
}

// UntrustProjectConfig removes the approval for path. It is idempotent.
func UntrustProjectConfig(path string) error {
	store, err := DefaultProjectTrustStore()
	if err != nil {
		return err
	}
	return store.Untrust(path)
}

// CheckProjectConfigTrust returns nil only when the currently inspected bytes
// are explicitly approved. It is suitable for non-interactive fail-closed use.
func CheckProjectConfigTrust(path string) error {
	store, err := DefaultProjectTrustStore()
	if err != nil {
		return err
	}
	return store.Check(path)
}

// Inspect fingerprints path and reports its current trust state without
// changing the store.
func (s *ProjectTrustStore) Inspect(path string) (ProjectTrustStatus, error) {
	status, _, err := s.inspectWithContent(path)
	return status, err
}

// Check returns nil only for an exact canonical-path and fingerprint match.
func (s *ProjectTrustStore) Check(path string) error {
	status, err := s.Inspect(path)
	if err != nil {
		return err
	}
	return trustError(status)
}

// Trust explicitly records the SHA-256 of the bytes currently at path. Calling
// it again for unchanged content is a no-op, including on disk.
func (s *ProjectTrustStore) Trust(path string) (ProjectTrustStatus, error) {
	projectTrustMu.Lock()
	defer projectTrustMu.Unlock()

	status, _, err := s.inspectWithContent(path)
	if err != nil {
		return status, err
	}
	if status.Trusted() {
		return status, nil
	}

	store, err := s.load()
	if err != nil {
		return status, err
	}
	trustedAt := time.Now().UTC().Format(time.RFC3339Nano)
	store.Projects[projectTrustKey(status.CanonicalPath)] = projectTrustEntry{
		CanonicalPath: status.CanonicalPath,
		Fingerprint:   status.Fingerprint,
		TrustedAt:     trustedAt,
	}
	if err := s.save(store); err != nil {
		return status, err
	}

	status.State = ProjectTrustTrusted
	status.TrustedFingerprint = status.Fingerprint
	status.TrustedAt = trustedAt
	return status, nil
}

// Untrust removes the approval for path. An already untrusted path succeeds
// without rewriting the store.
func (s *ProjectTrustStore) Untrust(path string) error {
	projectTrustMu.Lock()
	defer projectTrustMu.Unlock()

	canonicalPath, _, _, err := inspectProjectConfigFile(path)
	if err != nil {
		return err
	}
	store, err := s.load()
	if err != nil {
		return err
	}
	key := projectTrustKey(canonicalPath)
	if _, ok := store.Projects[key]; !ok {
		return nil
	}
	delete(store.Projects, key)
	return s.save(store)
}

func (s *ProjectTrustStore) inspectWithContent(path string) (ProjectTrustStatus, []byte, error) {
	canonicalPath, content, fingerprint, err := inspectProjectConfigFile(path)
	status := ProjectTrustStatus{
		CanonicalPath: canonicalPath,
		Fingerprint:   fingerprint,
		State:         ProjectTrustUntrusted,
	}
	if err != nil {
		return status, nil, err
	}
	store, err := s.load()
	if err != nil {
		return status, nil, err
	}
	entry, ok := store.Projects[projectTrustKey(canonicalPath)]
	if !ok {
		return status, content, nil
	}
	status.TrustedFingerprint = entry.Fingerprint
	status.TrustedAt = entry.TrustedAt
	if entry.Fingerprint == fingerprint {
		status.State = ProjectTrustTrusted
	} else {
		status.State = ProjectTrustChanged
	}
	return status, content, nil
}

func inspectProjectConfigFile(path string) (string, []byte, string, error) {
	canonicalPath, err := canonicalProjectConfigPath(path)
	if err != nil {
		return "", nil, "", err
	}
	content, err := readFileLimited(canonicalPath, maxTrustedProjectYAMLSize)
	if err != nil {
		return canonicalPath, nil, "", fmt.Errorf("read project configuration %s: %w", canonicalPath, err)
	}
	digest := sha256.Sum256(content)
	return canonicalPath, content, hex.EncodeToString(digest[:]), nil
}

func canonicalProjectConfigPath(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", errors.New("project configuration path is empty")
	}
	absPath, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", fmt.Errorf("make project configuration path absolute: %w", err)
	}
	canonicalPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		return "", fmt.Errorf("canonicalize project configuration %s: %w", absPath, err)
	}
	canonicalPath = filepath.Clean(canonicalPath)
	info, err := os.Stat(canonicalPath)
	if err != nil {
		return "", fmt.Errorf("stat project configuration %s: %w", canonicalPath, err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("project configuration is not a regular file: %s", canonicalPath)
	}
	return canonicalPath, nil
}

func projectTrustKey(canonicalPath string) string {
	key := filepath.Clean(canonicalPath)
	if runtime.GOOS == "windows" {
		key = strings.ToLower(key)
	}
	return key
}

func trustError(status ProjectTrustStatus) error {
	switch status.State {
	case ProjectTrustTrusted:
		return nil
	case ProjectTrustChanged:
		return &ProjectTrustError{Status: status, cause: ErrProjectConfigChanged}
	default:
		return &ProjectTrustError{Status: status, cause: ErrProjectConfigUntrusted}
	}
}

func (s *ProjectTrustStore) load() (projectTrustFile, error) {
	empty := projectTrustFile{Version: projectTrustStoreVersion, Projects: make(map[string]projectTrustEntry)}
	if s == nil || strings.TrimSpace(s.path) == "" {
		return empty, errors.New("project trust store path is empty")
	}
	data, err := readFileLimited(s.path, maxProjectTrustStoreSize)
	if errors.Is(err, os.ErrNotExist) {
		return empty, nil
	}
	if err != nil {
		return empty, fmt.Errorf("read project trust store %s: %w", s.path, err)
	}
	var store projectTrustFile
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&store); err != nil {
		return empty, fmt.Errorf("%w: decode %s: %v", ErrProjectTrustStoreCorrupt, s.path, err)
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return empty, fmt.Errorf("%w: decode %s: %v", ErrProjectTrustStoreCorrupt, s.path, err)
	}
	if err := validateProjectTrustFile(store); err != nil {
		return empty, fmt.Errorf("%w: %s: %v", ErrProjectTrustStoreCorrupt, s.path, err)
	}
	if store.Projects == nil {
		store.Projects = make(map[string]projectTrustEntry)
	}
	return store, nil
}

func readFileLimited(path string, maxBytes int64) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("not a regular file")
	}
	if info.Size() > maxBytes {
		return nil, fmt.Errorf("file exceeds %d bytes", maxBytes)
	}

	data, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("file exceeds %d bytes", maxBytes)
	}
	return data, nil
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

func validateProjectTrustFile(store projectTrustFile) error {
	if store.Version != projectTrustStoreVersion {
		return fmt.Errorf("unsupported version %d", store.Version)
	}
	for key, entry := range store.Projects {
		if !filepath.IsAbs(entry.CanonicalPath) || filepath.Clean(entry.CanonicalPath) != entry.CanonicalPath {
			return fmt.Errorf("entry %q has a non-canonical path", key)
		}
		if projectTrustKey(entry.CanonicalPath) != key {
			return fmt.Errorf("entry %q does not match canonical path", key)
		}
		digest, err := hex.DecodeString(entry.Fingerprint)
		if err != nil || len(digest) != sha256.Size {
			return fmt.Errorf("entry %q has an invalid SHA-256 fingerprint", key)
		}
		if _, err := time.Parse(time.RFC3339Nano, entry.TrustedAt); err != nil {
			return fmt.Errorf("entry %q has an invalid trust timestamp", key)
		}
	}
	return nil
}

func (s *ProjectTrustStore) save(store projectTrustFile) error {
	if err := validateProjectTrustFile(store); err != nil {
		return fmt.Errorf("refuse invalid project trust store: %w", err)
	}
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("encode project trust store: %w", err)
	}
	data = append(data, '\n')
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create project trust directory %s: %w", dir, err)
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return fmt.Errorf("restrict project trust directory %s: %w", dir, err)
	}
	if err := fsutil.WriteFileAtomic(s.path, data, 0o600); err != nil {
		return fmt.Errorf("write project trust store %s: %w", s.path, err)
	}
	if err := os.Chmod(s.path, 0o600); err != nil {
		return fmt.Errorf("restrict project trust store %s: %w", s.path, err)
	}
	return nil
}
