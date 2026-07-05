package env

import (
	"os"
	"regexp"
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

// maxExpandDepth caps %VAR% resolution recursion, breaking any reference cycle.
const maxExpandDepth = 10

// winVarRe matches Windows-style %NAME% environment references.
var winVarRe = regexp.MustCompile(`%([A-Za-z_][A-Za-z0-9_]*)%`)

// expandWindowsVars resolves %NAME% references in value. Each name is looked
// up first in the profile's own variables (profileEnv), then in the
// allow-listed system environment. Unknown names are left as the literal
// %NAME% (parity with Expand-RouterValue, code-router.ps1 L414-427). Chained
// references — a fusion profile sets OPENROUTER_API_KEY and then
// ANTHROPIC_AUTH_TOKEN=%OPENROUTER_API_KEY% — resolve recursively; depth caps
// any cycle. Resolution is order-independent (lookup over the whole map),
// unlike the PS line-by-line application, which is strictly more robust.
func expandWindowsVars(value string, profileEnv map[string]string, depth int) string {
	if depth <= 0 || !strings.Contains(value, "%") {
		return value
	}
	return winVarRe.ReplaceAllStringFunc(value, func(match string) string {
		name := match[1 : len(match)-1]
		if v, ok := profileEnv[name]; ok {
			return expandWindowsVars(v, profileEnv, depth-1)
		}
		if AllowedEnvVars[name] {
			if sys, ok := os.LookupEnv(name); ok {
				return sys
			}
		}
		return match // unresolved: keep the literal %NAME%, like PS
	})
}

// ExpandProfileEnv returns a copy of profileEnv with every %NAME% reference
// resolved (see expandWindowsVars). Used for both the cleaned environment and
// the CLEAR_ENV=false overlay so profiles behave identically in both modes.
func ExpandProfileEnv(profileEnv map[string]string) map[string]string {
	out := make(map[string]string, len(profileEnv))
	for k, v := range profileEnv {
		out[k] = expandWindowsVars(v, profileEnv, maxExpandDepth)
	}
	return out
}

// BuildCleanEnv returns a clean environment with only allowed system vars plus
// the profile vars, with %NAME% references in profile values resolved.
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

	// Overlay profile vars, resolving %NAME% references.
	for k, v := range ExpandProfileEnv(profileEnv) {
		env[k] = v
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
