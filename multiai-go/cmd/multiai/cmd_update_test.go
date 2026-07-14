package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/lrochetta/multiai/internal/update"
)

func restoreUpdateCommandHooks(t *testing.T) {
	t.Helper()
	originalFetch := fetchUpdateRelease
	originalInstall := installUpdateRelease
	originalVersion := version
	t.Cleanup(func() {
		fetchUpdateRelease = originalFetch
		installUpdateRelease = originalInstall
		version = originalVersion
	})
}

func TestParseUpdateFlagsRejectsUnknownInput(t *testing.T) {
	if _, err := parseUpdateFlags([]string{"--unknown"}); err == nil {
		t.Fatal("unknown option unexpectedly accepted")
	}
	if _, err := parseUpdateFlags([]string{"extra"}); err == nil {
		t.Fatal("positional argument unexpectedly accepted")
	}
}

func TestRunUpdateCheckNeverInstalls(t *testing.T) {
	restoreUpdateCommandHooks(t)
	version = "1.0.0"
	fetchUpdateRelease = func(context.Context) (*update.Release, error) {
		return &update.Release{TagName: "v1.1.0"}, nil
	}
	installed := false
	installUpdateRelease = func(context.Context, *update.Release) (*update.InstallResult, error) {
		installed = true
		return nil, nil
	}

	var output bytes.Buffer
	var errorOutput bytes.Buffer
	code := runUpdate([]string{"--check"}, strings.NewReader(""), &output, &errorOutput)
	if code != 0 {
		t.Fatalf("code=%d stderr=%q", code, errorOutput.String())
	}
	if installed {
		t.Fatal("--check invoked the installer")
	}
	if !strings.Contains(output.String(), `"has_update": true`) {
		t.Fatalf("JSON output = %q", output.String())
	}
}

func TestRunUpdateEOFDoesNotImplyConsent(t *testing.T) {
	restoreUpdateCommandHooks(t)
	version = "1.0.0"
	fetchUpdateRelease = func(context.Context) (*update.Release, error) {
		return &update.Release{TagName: "v1.1.0"}, nil
	}
	installed := false
	installUpdateRelease = func(context.Context, *update.Release) (*update.InstallResult, error) {
		installed = true
		return nil, nil
	}

	var output bytes.Buffer
	code := runUpdate(nil, strings.NewReader(""), &output, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("code=%d", code)
	}
	if installed {
		t.Fatal("EOF was treated as update consent")
	}
	if !strings.Contains(output.String(), "confirmation absente") {
		t.Fatalf("output = %q", output.String())
	}
}

func TestRunUpdateNeverClaimsSuccessWhenInstallFails(t *testing.T) {
	restoreUpdateCommandHooks(t)
	version = "1.0.0"
	fetchUpdateRelease = func(context.Context) (*update.Release, error) {
		return &update.Release{TagName: "v1.1.0"}, nil
	}
	installUpdateRelease = func(context.Context, *update.Release) (*update.InstallResult, error) {
		return nil, errors.New("npm failed")
	}

	var output bytes.Buffer
	var errorOutput bytes.Buffer
	code := runUpdate([]string{"--yes"}, strings.NewReader(""), &output, &errorOutput)
	if code != 2 {
		t.Fatalf("code=%d, want 2", code)
	}
	if strings.Contains(output.String(), "[OK]") || strings.Contains(output.String(), "installe de maniere persistante") {
		t.Fatalf("false success output = %q", output.String())
	}
	if !strings.Contains(errorOutput.String(), "non installee") {
		t.Fatalf("stderr = %q", errorOutput.String())
	}
}

func TestRunUpdateReportsOnlyVerifiedPersistentSuccess(t *testing.T) {
	restoreUpdateCommandHooks(t)
	t.Setenv("MULTIAI_CACHE_DIR", t.TempDir())
	version = "1.0.0"
	fetchUpdateRelease = func(context.Context) (*update.Release, error) {
		return &update.Release{TagName: "v1.1.0"}, nil
	}
	installUpdateRelease = func(context.Context, *update.Release) (*update.InstallResult, error) {
		return &update.InstallResult{
			Manager:        "npm",
			Version:        "1.1.0",
			ExecutablePath: "persistent/multiai",
		}, nil
	}

	var output bytes.Buffer
	var errorOutput bytes.Buffer
	code := runUpdate([]string{"--yes"}, strings.NewReader(""), &output, &errorOutput)
	if code != 0 {
		t.Fatalf("code=%d stderr=%q", code, errorOutput.String())
	}
	if !strings.Contains(output.String(), "[OK] multiai 1.1.0 installe de maniere persistante via npm") {
		t.Fatalf("output = %q", output.String())
	}
}

func TestRunUpdateUnsupportedInstallGivesSafeRecoveryCommand(t *testing.T) {
	restoreUpdateCommandHooks(t)
	version = "1.0.0"
	fetchUpdateRelease = func(context.Context) (*update.Release, error) {
		return &update.Release{TagName: "v1.1.0"}, nil
	}
	installUpdateRelease = func(context.Context, *update.Release) (*update.InstallResult, error) {
		return nil, update.ErrUnsupportedInstall
	}

	var errorOutput bytes.Buffer
	code := runUpdate([]string{"--yes"}, strings.NewReader(""), &bytes.Buffer{}, &errorOutput)
	if code != 2 {
		t.Fatalf("code=%d", code)
	}
	if !strings.Contains(errorOutput.String(), "npx --yes --allow-scripts=multiai multiai@1.1.0 install") {
		t.Fatalf("stderr = %q", errorOutput.String())
	}
}

func TestRunUpdateAlreadyCurrentSkipsInstall(t *testing.T) {
	restoreUpdateCommandHooks(t)
	version = "1.1.0"
	fetchUpdateRelease = func(context.Context) (*update.Release, error) {
		return &update.Release{TagName: "v1.1.0"}, nil
	}
	installed := false
	installUpdateRelease = func(context.Context, *update.Release) (*update.InstallResult, error) {
		installed = true
		return nil, nil
	}

	var output bytes.Buffer
	if code := runUpdate([]string{"--yes"}, strings.NewReader(""), &output, &bytes.Buffer{}); code != 0 {
		t.Fatalf("code=%d", code)
	}
	if installed || !strings.Contains(output.String(), "Deja a jour") {
		t.Fatalf("installed=%v output=%q", installed, output.String())
	}
}
