package dotenv

import (
	"strings"
	"testing"
)

func TestParse_Standard(t *testing.T) {
	input := "KEY1=value1\nKEY2=value2\n"
	result, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if result["KEY1"] != "value1" {
		t.Errorf("KEY1: got %q, want %q", result["KEY1"], "value1")
	}
	if result["KEY2"] != "value2" {
		t.Errorf("KEY2: got %q, want %q", result["KEY2"], "value2")
	}
}

func TestParse_Export(t *testing.T) {
	input := "export KEY1=value1\nexport KEY2=value2\n"
	result, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if result["KEY1"] != "value1" {
		t.Errorf("KEY1: got %q, want %q", result["KEY1"], "value1")
	}
}

func TestParse_ExportWithSpace(t *testing.T) {
	input := "export   KEY1=value1\n"
	result, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if result["KEY1"] != "value1" {
		t.Errorf("got %q, want %q", result["KEY1"], "value1")
	}
}

func TestParse_Comments(t *testing.T) {
	input := "# This is a comment\nKEY1=value1\n# Another comment\n"
	result, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 key, got %d", len(result))
	}
}

func TestParse_DoubleQuotes(t *testing.T) {
	input := `KEY1="value with spaces"`
	result, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if result["KEY1"] != "value with spaces" {
		t.Errorf("got %q, want %q", result["KEY1"], "value with spaces")
	}
}

func TestParse_SingleQuotes(t *testing.T) {
	input := "KEY1='value with spaces'"
	result, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if result["KEY1"] != "value with spaces" {
		t.Errorf("got %q, want %q", result["KEY1"], "value with spaces")
	}
}

func TestParse_EmptyLines(t *testing.T) {
	input := "\n\nKEY1=value1\n\n\n"
	result, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 key, got %d", len(result))
	}
}

func TestParse_EmptyFile(t *testing.T) {
	input := ""
	result, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 keys, got %d", len(result))
	}
}

func TestParse_MalformedLine(t *testing.T) {
	input := "NOT_A_VALID_LINE\nKEY1=value1\n"
	result, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 key, got %d", len(result))
	}
}

func TestParse_ValueWithEquals(t *testing.T) {
	input := "KEY1=value=with=equals\n"
	result, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if result["KEY1"] != "value=with=equals" {
		t.Errorf("got %q, want %q", result["KEY1"], "value=with=equals")
	}
}

func TestIsPlaceholder(t *testing.T) {
	tests := []struct {
		value    string
		expected bool
	}{
		{"", true},
		{"   ", true},
		{"PASTE_YOUR_KEY_HERE", true},
		{"YOUR_API_KEY", true},
		{"sk-xxxx", true},
		{"TODO", true},
		{"replace_me", true},
		{"change_me", true},
		{"ta_cle_ici", true},
		{"xxx", true},
		{"MY_VAR_HERE", true},
		{"sk-ant-api-03-abc123def456", false},
		{"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.xxx", false},
		{"dGVzdC10b2tlbg==", false},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			result := IsPlaceholder(tt.value)
			if result != tt.expected {
				t.Errorf("IsPlaceholder(%q) = %v, want %v", tt.value, result, tt.expected)
			}
		})
	}
}
