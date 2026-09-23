package runners

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRemoveExtension(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{"simple extension", "photo.png", "photo"},
		{"no extension", "README", "README"},
		{"multiple dots", "archive.tar.gz", "archive.tar"},
		{"leading dot file", ".gitignore", ""},
		{"trailing dot", "file.", "file"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := removeExtension(tt.filename); got != tt.want {
				t.Fatalf("removeExtension(%q) = %q, want %q", tt.filename, got, tt.want)
			}
		})
	}
}

func TestSha256sum(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.txt")
	content := []byte("hello world")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	got, err := sha256sum(path)
	if err != nil {
		t.Fatalf("sha256sum returned error: %v", err)
	}
	// sha256("hello world") in hex.
	const want = "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	if got != want {
		t.Fatalf("sha256sum = %q, want %q", got, want)
	}
}

func TestSha256sumMissingFile(t *testing.T) {
	if _, err := sha256sum(filepath.Join(t.TempDir(), "does-not-exist")); err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}
