//go:build windows

package secret

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

// Windows Credential Manager constants.
const (
	credTypeGeneric         = 1
	credPersistLocalMachine = 2
)

// Windows error codes.
const errorNotFound = syscall.Errno(1168) // ERROR_NOT_FOUND

// CREDENTIALW matches the Win32 CREDENTIALW structure layout on 64-bit Windows.
//
//	https://learn.microsoft.com/en-us/windows/win32/api/wincred/ns-wincred-credentialw
type CREDENTIALW struct {
	Flags              uint32
	Type               uint32
	TargetName         *uint16
	Comment            *uint16
	LastWritten        filetime
	CredentialBlobSize uint32
	_                  uint32 // padding for 8-byte pointer alignment
	CredentialBlob     *byte
	Persist            uint32
	AttributeCount     uint32
	Attributes         uintptr
	TargetAlias        *uint16
	UserName           *uint16
}

// filetime matches the Win32 FILETIME structure.
type filetime struct {
	LowDateTime  uint32
	HighDateTime uint32
}

// Lazy-loaded advapi32.dll function pointers.
var (
	advapi32 = syscall.NewLazyDLL("advapi32.dll")

	procCredWriteW     = advapi32.NewProc("CredWriteW")
	procCredReadW      = advapi32.NewProc("CredReadW")
	procCredDeleteW    = advapi32.NewProc("CredDeleteW")
	procCredEnumerateW = advapi32.NewProc("CredEnumerateW")
	procCredFree       = advapi32.NewProc("CredFree")
)

// winCredStore implements Store using the Windows Credential Manager.
type winCredStore struct{}

// newPlatformStore returns a Windows Credential Manager-backed Store.
//
// Falls back to the encrypted file store when MULTIAI_SECRETS_DIR is set,
// preserving existing test behaviour and portable-mode operation.
func newPlatformStore() (Store, error) {
	if os.Getenv("MULTIAI_SECRETS_DIR") != "" {
		return newEncryptedFileStore()
	}
	return &winCredStore{}, nil
}

// newNamedStore returns the requested named backend on Windows.
func newNamedStore(backend string) (Store, error) {
	switch backend {
	case "wincred":
		return newPlatformStore()
	default:
		return nil, fmt.Errorf("unsupported backend on this platform: %s (supported: wincred, file, auto)", backend)
	}
}

// ── Store interface ────────────────────────────────────────────────────

func (s *winCredStore) Get(service, key string) (string, error) {
	pcred, err := readCredential(targetName(service, key))
	if err != nil {
		if err == errorNotFound {
			return "", fmt.Errorf("credential not found: %s/%s", service, key)
		}
		return "", fmt.Errorf("CredReadW: %w", err)
	}
	defer credFree(unsafe.Pointer(pcred))

	blob := unsafe.Slice(pcred.CredentialBlob, int(pcred.CredentialBlobSize))
	val := make([]byte, len(blob))
	copy(val, blob)
	return string(val), nil
}

func (s *winCredStore) Set(service, key, value string) error {
	tn, err := syscall.UTF16PtrFromString(targetName(service, key))
	if err != nil {
		return err
	}

	blob := []byte(value)
	var blobPtr *byte
	blobSize := uint32(len(blob))
	if blobSize > 0 {
		blobPtr = &blob[0]
	}

	cred := &CREDENTIALW{
		Type:               credTypeGeneric,
		TargetName:         tn,
		CredentialBlobSize: blobSize,
		CredentialBlob:     blobPtr,
		Persist:            credPersistLocalMachine,
	}

	ret, _, err := procCredWriteW.Call(
		uintptr(unsafe.Pointer(cred)),
		0, // reserved
	)
	if ret == 0 {
		return fmt.Errorf("CredWriteW: %w", err)
	}
	return nil
}

func (s *winCredStore) Delete(service, key string) error {
	tn, err := syscall.UTF16PtrFromString(targetName(service, key))
	if err != nil {
		return err
	}

	ret, _, err := procCredDeleteW.Call(
		uintptr(unsafe.Pointer(tn)),
		uintptr(credTypeGeneric),
		0, // reserved
	)
	if ret == 0 && err != errorNotFound {
		return fmt.Errorf("CredDeleteW: %w", err)
	}
	return nil
}

func (s *winCredStore) List(service string) (map[string]string, error) {
	filter, err := syscall.UTF16PtrFromString(serviceFilter(service))
	if err != nil {
		return nil, err
	}

	var count uint32
	var pCredentials **CREDENTIALW

	ret, _, err := procCredEnumerateW.Call(
		uintptr(unsafe.Pointer(filter)),
		0, // Flags
		uintptr(unsafe.Pointer(&count)),
		uintptr(unsafe.Pointer(&pCredentials)),
	)
	if ret == 0 {
		if err == errorNotFound {
			return make(map[string]string), nil
		}
		return nil, fmt.Errorf("CredEnumerateW: %w", err)
	}
	defer credFree(unsafe.Pointer(pCredentials))

	if count == 0 {
		return make(map[string]string), nil
	}

	creds := make(map[string]string, count)
	for _, pcred := range unsafe.Slice(pCredentials, int(count)) {
		key := extractKey(utf16PtrToString(pcred.TargetName))
		if key == "" {
			continue
		}
		blob := unsafe.Slice(pcred.CredentialBlob, int(pcred.CredentialBlobSize))
		val := make([]byte, len(blob))
		copy(val, blob)
		creds[key] = string(val)
	}
	return creds, nil
}

// ── Win32 helpers ──────────────────────────────────────────────────────

// readCredential calls CredReadW and returns the CREDENTIALW pointer.
// The caller MUST call credFree on the returned pointer.
func readCredential(targetName string) (*CREDENTIALW, error) {
	tn, err := syscall.UTF16PtrFromString(targetName)
	if err != nil {
		return nil, err
	}

	var pcred *CREDENTIALW
	ret, _, err := procCredReadW.Call(
		uintptr(unsafe.Pointer(tn)),
		uintptr(credTypeGeneric),
		0, // reserved
		uintptr(unsafe.Pointer(&pcred)),
	)
	if ret == 0 {
		return nil, err
	}
	return pcred, nil
}

// credFree calls CredFree to release memory allocated by CredReadW or
// CredEnumerateW.
func credFree(p unsafe.Pointer) {
	procCredFree.Call(uintptr(p))
}

// utf16PtrToString converts a null-terminated UTF-16 pointer to a Go string.
// The source memory is C-allocated (by the Credential Manager API) and is not
// subject to Go GC movement, so creating a fixed-size slice is safe.
func utf16PtrToString(p *uint16) string {
	if p == nil {
		return ""
	}
	// Max key length in practice is < 64 chars; 4096 is far beyond any
	// conceivable credential target name.
	const maxLen = 4096
	slice := unsafe.Slice(p, maxLen)
	n := 0
	for n < maxLen && slice[n] != 0 {
		n++
	}
	if n == 0 {
		return ""
	}
	return syscall.UTF16ToString(slice[:n])
}
