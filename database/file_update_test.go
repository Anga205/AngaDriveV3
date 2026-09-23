package database

import (
	"strings"
	"testing"
)

func TestValidateFileName(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantErr  bool
	}{
		{"valid simple name", "report.pdf", false},
		{"valid name with spaces", "my report final v2.pdf", false},
		{"valid unicode name", "résumé.pdf", false},
		{"empty name", "", true},
		{"whitespace only", "   ", true},
		{"dot", ".", true},
		{"double dot", "..", true},
		{"too long", string(make([]byte, 256)), true},
		{"contains forward slash", "a/b.txt", true},
		{"contains backslash", `a\b.txt`, true},
		{"contains control char", "a\x00b.txt", true},
		{"newline in name", "a\nb.txt", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFileName(tt.filename)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateFileName(%q) error = %v, wantErr %v", tt.filename, err, tt.wantErr)
			}
		})
	}
}

func TestValidateFileNameAcceptsMaxLength(t *testing.T) {
	// Exactly 255 bytes is the boundary; it should be accepted.
	filename := strings.Repeat("a", 255)
	if err := validateFileName(filename); err != nil {
		t.Fatalf("expected 255-byte name to be valid, got error: %v", err)
	}
}
