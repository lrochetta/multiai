//go:build darwin && !cgo

package secret

import (
	"fmt"
	"os/exec"
	"strings"
)

func keychainAvailable() bool {
	_, err := exec.LookPath("/usr/bin/security")
	return err == nil
}

func keychainGet(service, key string) (string, error) {
	cmd := exec.Command("/usr/bin/security",
		"find-generic-password",
		"-s", service,
		"-a", key,
		"-w")
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if asExit(err, &exitErr) && exitErr.ExitCode() == 44 {
			return "", fmt.Errorf("credential not found: %s/%s", service, key)
		}
		return "", fmt.Errorf("keychain get %s/%s: %w", service, key, err)
	}
	return strings.TrimRight(string(out), "\n\r"), nil
}

func keychainSet(service, key, value string) error {
	cmd := exec.Command("/usr/bin/security",
		"add-generic-password",
		"-s", service,
		"-a", key,
		"-w", value,
		"-U") // -U: update existing item if present
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("keychain set %s/%s: %w", service, key, err)
	}
	return nil
}

func keychainDelete(service, key string) error {
	cmd := exec.Command("/usr/bin/security",
		"delete-generic-password",
		"-s", service,
		"-a", key)
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if asExit(err, &exitErr) && exitErr.ExitCode() == 44 {
			// Item not found — idempotent delete is fine.
			return nil
		}
		return fmt.Errorf("keychain delete %s/%s: %w", service, key, err)
	}
	return nil
}

func keychainList(service string) (map[string]string, error) {
	// 1. Dump all generic passwords and parse to find accounts for this service.
	cmd := exec.Command("/usr/bin/security", "dump-keychain", "-r", "genp")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("keychain dump-keychain: %w", err)
	}

	entries := parseDumpKeychain(string(out))
	accounts := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.Service == service {
			accounts = append(accounts, e.Account)
		}
	}
	if len(accounts) == 0 {
		return make(map[string]string), nil
	}

	// 2. Fetch each password individually.
	result := make(map[string]string, len(accounts))
	for _, acct := range accounts {
		cmd := exec.Command("/usr/bin/security",
			"find-generic-password",
			"-s", service,
			"-a", acct,
			"-w")
		pw, err := cmd.Output()
		if err != nil {
			// Skip entries that can't be read — they may be in a locked
			// keychain or require user interaction.
			continue
		}
		result[acct] = strings.TrimRight(string(pw), "\n\r")
	}
	return result, nil
}

// asExit extracts an *exec.ExitError from err (supports unwrapping).
func asExit(err error, target **exec.ExitError) bool {
	if err == nil {
		return false
	}
	e, ok := err.(*exec.ExitError)
	if ok {
		*target = e
		return true
	}
	// Try unwrapping via interface{ Unwrap() error }.
	type unwrapper interface{ Unwrap() error }
	if u, ok := err.(unwrapper); ok {
		return asExit(u.Unwrap(), target)
	}
	return false
}
