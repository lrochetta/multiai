package fsutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFileAtomic_Success(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	data := []byte("hello atomic write")

	if err := WriteFileAtomic(path, data, 0644); err != nil {
		t.Fatalf("WriteFileAtomic failed: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(data) {
		t.Errorf("content = %q, want %q", string(got), string(data))
	}
}

func TestWriteFileAtomic_PermissionDenied(t *testing.T) {
	// Point to a path whose parent directory does not exist.
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent", "file.txt")

	err := WriteFileAtomic(path, []byte("data"), 0644)
	if err == nil {
		t.Fatal("expected error for non-existent parent directory, got nil")
	}
}

func TestWriteFileAtomic_NoPartialWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "target.txt")
	original := []byte("original content before crash")

	if err := os.WriteFile(path, original, 0644); err != nil {
		t.Fatal(err)
	}

	// Make the target file read-only so that os.Rename (called by
	// WriteFileAtomic) cannot replace it. On Unix this sets the file
	// mode; on Windows it sets FILE_ATTRIBUTE_READONLY, which causes
	// MoveFileEx (backing os.Rename) to fail with ERROR_ACCESS_DENIED.
	if err := os.Chmod(path, 0444); err != nil {
		t.Skipf("chmod not supported on this platform: %v", err)
	}

	newData := []byte("new content that should never reach the file")
	err := WriteFileAtomic(path, newData, 0644)
	if err == nil {
		t.Fatal("expected error due to read-only target, got nil")
	}

	// Restore permissions so we can read back and verify the content.
	_ = os.Chmod(path, 0644)

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(original) {
		t.Errorf("original file was modified or corrupted: got %q, want %q", string(got), string(original))
	}
}
