package secret

import (
	"fmt"
	"os"
	"sync"

	"github.com/lrochetta/multiai/internal/i18n"
)

// fallbackStoreWrapper wraps an encryptedFileStore and emits a one-time
// warning when the native credential store was unavailable. It satisfies
// the Store interface so callers that received it from newPlatformStore()
// do not need to distinguish between native and file-backed stores.
type fallbackStoreWrapper struct {
	store *encryptedFileStore
	once  sync.Once
}

// warn prints a fallback warning to stderr exactly once per process lifetime.
func (w *fallbackStoreWrapper) warn() {
	w.once.Do(func() {
		fmt.Fprintf(os.Stderr, "[!] %s\n", i18n.T("store_fallback"))
	})
}

func (w *fallbackStoreWrapper) Get(service, key string) (string, error) {
	w.warn()
	return w.store.Get(service, key)
}

func (w *fallbackStoreWrapper) Set(service, key, value string) error {
	w.warn()
	return w.store.Set(service, key, value)
}

func (w *fallbackStoreWrapper) Delete(service, key string) error {
	w.warn()
	return w.store.Delete(service, key)
}

func (w *fallbackStoreWrapper) List(service string) (map[string]string, error) {
	w.warn()
	return w.store.List(service)
}
