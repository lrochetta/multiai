package env

import (
	"os"
	"strings"
)

// AllowedEnvVars are the only environment variables kept when clearing.
var AllowedEnvVars = map[string]bool{
	"PATH": true, "PATHEXT": true, "HOME": true, "USER": true,
	"USERPROFILE": true, "USERNAME": true,
	"TEMP": true, "TMP": true, "TMPDIR": true,
	"SHELL": true, "LANG": true, "LC_ALL": true, "LC_CTYPE": true,
	"DISPLAY": true, "WAYLAND_DISPLAY": true,
	"TERM": true, "COLORTERM": true,
	"SSH_AUTH_SOCK": true, "SSH_AGENT_PID": true,
	"SYSTEMROOT": true, "WINDIR": true, "COMSPEC": true,
	"OS": true, "PROCESSOR_ARCHITECTURE": true,
	"LOGNAME": true, "PWD": true, "OLDPWD": true,
	"XDG_SESSION_TYPE": true, "DBUS_SESSION_BUS_ADDRESS": true,
}

// BuildCleanEnv returns a clean environment with only allowed vars + profile vars.
func BuildCleanEnv(profileEnv map[string]string) []string {
	// Start with allowed system vars
	env := make(map[string]string)
	for _, kv := range os.Environ() {
		idx := strings.Index(kv, "=")
		if idx < 0 {
			continue
		}
		key := kv[:idx]
		value := kv[idx+1:]
		if AllowedEnvVars[key] {
			env[key] = value
		}
	}

	// Overlay profile vars
	for k, v := range profileEnv {
		env[k] = os.ExpandEnv(v)
	}

	// Convert to []string format
	var result []string
	for k, v := range env {
		result = append(result, k+"="+v)
	}
	return result
}

// IsSecretKey returns true if the key name suggests it contains a secret.
func IsSecretKey(key string) bool {
	upper := strings.ToUpper(key)
	for _, pattern := range []string{"KEY", "TOKEN", "SECRET", "PASSWORD", "AUTH", "CREDENTIAL"} {
		if strings.Contains(upper, pattern) {
			return true
		}
	}
	return false
}

// MaskSecret masks a secret value for display.
func MaskSecret(value string) string {
	if len(value) > 8 {
		return value[:4] + "..." + value[len(value)-4:]
	}
	if len(value) > 0 {
		return "***"
	}
	return "<vide>"
}
