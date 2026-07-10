//go:build darwin && cgo

package secret

/*
#cgo LDFLAGS: -framework Security -framework CoreFoundation

#include <Security/Security.h>
#include <CoreFoundation/CoreFoundation.h>
#include <stdlib.h>
#include <string.h>

// ── CF helpers ──────────────────────────────────────────────────────────

static CFStringRef _cfstr(const char *s) {
	return CFStringCreateWithCString(kCFAllocDefault, s, kCFStringEncodingUTF8);
}

// _cfstrdup returns a malloc'd C copy of the CFString, or NULL.
static char *_cfstrdup(CFStringRef s) {
	if (!s) return NULL;
	CFIndex len = CFStringGetLength(s);
	CFIndex max = CFStringGetMaximumSizeForEncoding(len, kCFStringEncodingUTF8) + 1;
	char *buf = (char *)malloc((size_t)max);
	if (buf && !CFStringGetCString(s, buf, max, kCFStringEncodingUTF8)) {
		free(buf);
		return NULL;
	}
	return buf;
}

// ── Set (add-or-update) ────────────────────────────────────────────────
//
// Returns 0 on success, -1 on error (with *error set to a malloc'd string
// the caller must free).

int darwin_keychain_set(const char *service, const char *account,
						const char *password, char **error) {
	CFStringRef cfService = _cfstr(service);
	CFStringRef cfAccount = _cfstr(account);
	CFDataRef cfPassword = CFDataCreate(kCFAllocDefault,
		(const UInt8 *)password, (CFIndex)strlen(password));

	const void *addKeys[] = { kSecClass, kSecAttrService, kSecAttrAccount, kSecValueData };
	const void *addVals[] = { kSecClassGenericPassword, cfService, cfAccount, cfPassword };
	CFDictionaryRef addQuery = CFDictionaryCreate(kCFAllocDefault,
		addKeys, addVals, 4,
		&kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);

	OSStatus status = SecItemAdd(addQuery, NULL);
	if (status == errSecDuplicateItem) {
		// Update the existing item's password.
		const void *findKeys[] = { kSecClass, kSecAttrService, kSecAttrAccount };
		const void *findVals[] = { kSecClassGenericPassword, cfService, cfAccount };
		CFDictionaryRef findQuery = CFDictionaryCreate(kCFAllocDefault,
			findKeys, findVals, 3,
			&kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);

		const void *updKeys[] = { kSecValueData };
		const void *updVals[] = { cfPassword };
		CFDictionaryRef updQuery = CFDictionaryCreate(kCFAllocDefault,
			updKeys, updVals, 1,
			&kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);

		status = SecItemUpdate(findQuery, updQuery);
		CFRelease(updQuery);
		CFRelease(findQuery);
	}

	CFRelease(addQuery);
	CFRelease(cfPassword);
	CFRelease(cfAccount);
	CFRelease(cfService);

	if (status != errSecSuccess) {
		if (error) *error = strdup("Security Framework call failed");
		return -1;
	}
	return 0;
}

// ── Get ────────────────────────────────────────────────────────────────
//
// Returns 0 on success (item found: *password is a malloc'd C string the
// caller must free; not found: *password is NULL).  Returns -1 on error.

int darwin_keychain_get(const char *service, const char *account,
						char **password, char **error) {
	CFStringRef cfService = _cfstr(service);
	CFStringRef cfAccount = _cfstr(account);

	const void *keys[] = { kSecClass, kSecAttrService, kSecAttrAccount,
						   kSecReturnData, kSecMatchLimit };
	const void *vals[] = { kSecClassGenericPassword, cfService, cfAccount,
						   kCFBooleanTrue, kSecMatchLimitOne };
	CFDictionaryRef query = CFDictionaryCreate(kCFAllocDefault,
		keys, vals, 5,
		&kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);

	CFTypeRef result = NULL;
	OSStatus status = SecItemCopyMatching(query, &result);
	CFRelease(query);
	CFRelease(cfAccount);
	CFRelease(cfService);

	if (status == errSecItemNotFound) {
		*password = NULL;
		return 0;
	}
	if (status != errSecSuccess) {
		if (error) *error = strdup("Security Framework call failed");
		return -1;
	}

	CFDataRef data = (CFDataRef)result;
	CFIndex len = CFDataGetLength(data);
	const UInt8 *bytes = CFDataGetBytePtr(data);
	*password = (char *)malloc((size_t)len + 1);
	if (*password) {
		memcpy(*password, bytes, (size_t)len);
		(*password)[len] = '\0';
	}
	CFRelease(result);
	return 0;
}

// ── Delete ─────────────────────────────────────────────────────────────
//
// Idempotent: succeeds (returns 0) even when the item does not exist.

int darwin_keychain_delete(const char *service, const char *account,
						   char **error) {
	CFStringRef cfService = _cfstr(service);
	CFStringRef cfAccount = _cfstr(account);

	const void *keys[] = { kSecClass, kSecAttrService, kSecAttrAccount };
	const void *vals[] = { kSecClassGenericPassword, cfService, cfAccount };
	CFDictionaryRef query = CFDictionaryCreate(kCFAllocDefault,
		keys, vals, 3,
		&kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);

	OSStatus status = SecItemDelete(query);
	CFRelease(query);
	CFRelease(cfAccount);
	CFRelease(cfService);

	if (status != errSecSuccess && status != errSecItemNotFound) {
		if (error) *error = strdup("Security Framework call failed");
		return -1;
	}
	return 0;
}

// ── List ───────────────────────────────────────────────────────────────
//
// Returns all (account, password) pairs for the given service.
// On success *count is set and *accounts / *passwords are malloc'd arrays
// the caller must free via darwin_keychain_free_strings.

int darwin_keychain_list(const char *service, char ***accounts,
						 char ***passwords, int *count, char **error) {
	CFStringRef cfService = _cfstr(service);

	const void *keys[] = { kSecClass, kSecAttrService,
						   kSecReturnAttributes, kSecReturnData, kSecMatchLimit };
	const void *vals[] = { kSecClassGenericPassword, cfService,
						   kCFBooleanTrue, kCFBooleanTrue, kSecMatchLimitAll };
	CFDictionaryRef query = CFDictionaryCreate(kCFAllocDefault,
		keys, vals, 5,
		&kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);

	CFTypeRef result = NULL;
	OSStatus status = SecItemCopyMatching(query, &result);
	CFRelease(query);

	if (status == errSecItemNotFound) {
		*count = 0;
		*accounts = NULL;
		*passwords = NULL;
		CFRelease(cfService);
		return 0;
	}
	if (status != errSecSuccess) {
		if (error) *error = strdup("Security Framework call failed");
		CFRelease(cfService);
		return -1;
	}

	CFArrayRef array = (CFArrayRef)result;
	CFIndex n = CFArrayGetCount(array);
	*count = (int)n;
	*accounts = (char **)calloc((size_t)n, sizeof(char *));
	*passwords = (char **)calloc((size_t)n, sizeof(char *));

	for (CFIndex i = 0; i < n; i++) {
		CFDictionaryRef item = (CFDictionaryRef)CFArrayGetValueAtIndex(array, i);

		CFStringRef acct = (CFStringRef)CFDictionaryGetValue(item, kSecAttrAccount);
		(*accounts)[i] = acct ? _cfstrdup(acct) : strdup("");

		CFDataRef data = (CFDataRef)CFDictionaryGetValue(item, kSecValueData);
		if (data) {
			CFIndex dlen = CFDataGetLength(data);
			const UInt8 *dbytes = CFDataGetBytePtr(data);
			(*passwords)[i] = (char *)malloc((size_t)dlen + 1);
			if ((*passwords)[i]) {
				memcpy((*passwords)[i], dbytes, (size_t)dlen);
				(*passwords)[i][dlen] = '\0';
			}
		} else {
			(*passwords)[i] = strdup("");
		}
	}

	CFRelease(array);
	CFRelease(cfService);
	return 0;
}

// ── Free helpers ───────────────────────────────────────────────────────

void darwin_keychain_free_strings(char **strings, int count) {
	if (!strings) return;
	for (int i = 0; i < count; i++) {
		if (strings[i]) free(strings[i]);
	}
	free(strings);
}
*/
import "C"
import (
	"fmt"
	"unsafe"
)

func keychainAvailable() bool {
	return true
}

func keychainGet(service, key string) (string, error) {
	cService := C.CString(service)
	cKey := C.CString(key)
	defer C.free(unsafe.Pointer(cService))
	defer C.free(unsafe.Pointer(cKey))

	var cPassword *C.char
	var cError *C.char

	ret := C.darwin_keychain_get(cService, cKey, &cPassword, &cError)
	if ret != 0 {
		errMsg := ""
		if cError != nil {
			errMsg = C.GoString(cError)
			C.free(unsafe.Pointer(cError))
		} else {
			errMsg = "unknown error"
		}
		return "", fmt.Errorf("keychain get %s/%s: %s", service, key, errMsg)
	}
	if cPassword == nil {
		return "", fmt.Errorf("credential not found: %s/%s", service, key)
	}
	password := C.GoString(cPassword)
	C.free(unsafe.Pointer(cPassword))
	return password, nil
}

func keychainSet(service, key, value string) error {
	cService := C.CString(service)
	cKey := C.CString(key)
	cValue := C.CString(value)
	defer C.free(unsafe.Pointer(cService))
	defer C.free(unsafe.Pointer(cKey))
	defer C.free(unsafe.Pointer(cValue))

	var cError *C.char
	ret := C.darwin_keychain_set(cService, cKey, cValue, &cError)
	if ret != 0 {
		errMsg := ""
		if cError != nil {
			errMsg = C.GoString(cError)
			C.free(unsafe.Pointer(cError))
		} else {
			errMsg = "unknown error"
		}
		return fmt.Errorf("keychain set %s/%s: %s", service, key, errMsg)
	}
	return nil
}

func keychainDelete(service, key string) error {
	cService := C.CString(service)
	cKey := C.CString(key)
	defer C.free(unsafe.Pointer(cService))
	defer C.free(unsafe.Pointer(cKey))

	var cError *C.char
	ret := C.darwin_keychain_delete(cService, cKey, &cError)
	if ret != 0 {
		errMsg := ""
		if cError != nil {
			errMsg = C.GoString(cError)
			C.free(unsafe.Pointer(cError))
		} else {
			errMsg = "unknown error"
		}
		return fmt.Errorf("keychain delete %s/%s: %s", service, key, errMsg)
	}
	return nil
}

func keychainList(service string) (map[string]string, error) {
	cService := C.CString(service)
	defer C.free(unsafe.Pointer(cService))

	var cAccounts, cPasswords **C.char
	var cCount C.int
	var cError *C.char

	ret := C.darwin_keychain_list(cService, &cAccounts, &cPasswords, &cCount, &cError)
	if ret != 0 {
		errMsg := ""
		if cError != nil {
			errMsg = C.GoString(cError)
			C.free(unsafe.Pointer(cError))
		} else {
			errMsg = "unknown error"
		}
		return nil, fmt.Errorf("keychain list %s: %s", service, errMsg)
	}

	count := int(cCount)
	if count == 0 {
		return make(map[string]string), nil
	}

	// Build the result map from C arrays.
	result := make(map[string]string, count)
	accts := (*[1 << 28]*C.char)(unsafe.Pointer(cAccounts))[:count:count]
	pws := (*[1 << 28]*C.char)(unsafe.Pointer(cPasswords))[:count:count]

	for i := 0; i < count; i++ {
		acct := C.GoString(accts[i])
		pw := C.GoString(pws[i])
		result[acct] = pw
	}

	// Free C arrays.
	C.darwin_keychain_free_strings(cAccounts, cCount)
	C.darwin_keychain_free_strings(cPasswords, cCount)

	return result, nil
}
