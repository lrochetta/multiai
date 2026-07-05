package menu

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseBmadConfig(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantVersion string
		wantPacks   []string
	}{
		{
			name:        "version and packs",
			content:     "bmad_version: 0.7.5\npacks:\n  - core\n  - agents\nother: x\n",
			wantVersion: "0.7.5",
			wantPacks:   []string{"core", "agents"},
		},
		{
			name:        "version only",
			content:     "bmad_version: 1.0.0\n",
			wantVersion: "1.0.0",
			wantPacks:   nil,
		},
		{
			name:        "packs block stops at first non-indented line",
			content:     "packs:\n  - core\ntop_level: yes\n  - not-a-pack\n",
			wantVersion: "",
			wantPacks:   []string{"core"},
		},
		{
			name:        "blank lines inside packs block are skipped",
			content:     "packs:\n  - core\n\n  - extra\n",
			wantVersion: "",
			wantPacks:   []string{"core", "extra"},
		},
		{
			name:        "crlf line endings",
			content:     "bmad_version: 0.9.1\r\npacks:\r\n  - core\r\n",
			wantVersion: "0.9.1",
			wantPacks:   []string{"core"},
		},
		{
			name:        "empty content",
			content:     "",
			wantVersion: "",
			wantPacks:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version, packs := parseBmadConfig(tt.content)
			if version != tt.wantVersion {
				t.Errorf("version = %q, want %q", version, tt.wantVersion)
			}
			if !reflect.DeepEqual(packs, tt.wantPacks) {
				t.Errorf("packs = %v, want %v", packs, tt.wantPacks)
			}
		})
	}
}

func TestDetectBmad(t *testing.T) {
	writeFile := func(t *testing.T, path, content string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	tests := []struct {
		name  string
		setup func(t *testing.T, dir string)
		want  bmadInfo
	}{
		{
			name: "bmad config yaml with version and packs",
			setup: func(t *testing.T, dir string) {
				writeFile(t, filepath.Join(dir, "_bmad", "config.yaml"),
					"bmad_version: 0.7.5\npacks:\n  - core\n")
			},
			want: bmadInfo{Installed: true, Version: "0.7.5", Packs: []string{"core"}},
		},
		{
			name: "package json devDependencies",
			setup: func(t *testing.T, dir string) {
				writeFile(t, filepath.Join(dir, "package.json"),
					`{"devDependencies": {"bmad-plus": "^0.8.0"}}`)
			},
			want: bmadInfo{Installed: true, Version: "^0.8.0"},
		},
		{
			name: "package json without bmad-plus",
			setup: func(t *testing.T, dir string) {
				writeFile(t, filepath.Join(dir, "package.json"),
					`{"devDependencies": {"typescript": "^5.0.0"}}`)
			},
			want: bmadInfo{},
		},
		{
			name: "invalid package json is ignored",
			setup: func(t *testing.T, dir string) {
				writeFile(t, filepath.Join(dir, "package.json"), "{not json")
			},
			want: bmadInfo{},
		},
		{
			name: "agents directory only",
			setup: func(t *testing.T, dir string) {
				if err := os.MkdirAll(filepath.Join(dir, ".agents"), 0o755); err != nil {
					t.Fatal(err)
				}
			},
			want: bmadInfo{Installed: true},
		},
		{
			name: "agents as plain file does not count",
			setup: func(t *testing.T, dir string) {
				writeFile(t, filepath.Join(dir, ".agents"), "not a dir")
			},
			want: bmadInfo{},
		},
		{
			name: "config yaml wins over package json",
			setup: func(t *testing.T, dir string) {
				writeFile(t, filepath.Join(dir, "_bmad", "config.yaml"), "bmad_version: 1.0.0\n")
				writeFile(t, filepath.Join(dir, "package.json"),
					`{"devDependencies": {"bmad-plus": "^0.8.0"}}`)
			},
			want: bmadInfo{Installed: true, Version: "1.0.0"},
		},
		{
			name:  "empty directory",
			setup: func(t *testing.T, dir string) {},
			want:  bmadInfo{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setup(t, dir)
			got := detectBmad(dir)
			if got.Installed != tt.want.Installed || got.Version != tt.want.Version ||
				!reflect.DeepEqual(got.Packs, tt.want.Packs) {
				t.Errorf("detectBmad() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestNpmVersionValidation(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"0.7.5", true},
		{"latest", true},
		{"0.8.0-rc.1", true},
		{"1", true},
		{"", false},
		{"-pre", false},
		{"0.7.5 && del *", false},
		{`0.7.5" --evil`, false},
		{"0.7.5%PATH%", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := npmVersionRe.MatchString(tt.input); got != tt.valid {
				t.Errorf("npmVersionRe.MatchString(%q) = %v, want %v", tt.input, got, tt.valid)
			}
		})
	}
}
