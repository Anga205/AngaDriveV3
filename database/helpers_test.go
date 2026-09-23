package database

import (
	"errors"
	"testing"

	"gorm.io/gorm"
)

func TestIsDuplicateKeyError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"gorm duplicate key", gorm.ErrDuplicatedKey, true},
		{"sqlite unique constraint", errors.New("UNIQUE constraint failed: collection_files.collection_id"), true},
		{"generic constraint", errors.New("constraint failed"), true},
		{"duplicate keyword", errors.New("duplicate entry"), true},
		{"unrelated error", errors.New("some other error"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isDuplicateKeyError(tt.err); got != tt.want {
				t.Fatalf("isDuplicateKeyError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestCollectionGetEditors(t *testing.T) {
	tests := []struct {
		name    string
		editors string
		want    []string
	}{
		{"single editor", "token1", []string{"token1"}},
		{"multiple editors", "token1,token2,token3", []string{"token1", "token2", "token3"}},
		{"editors with spaces", "token1, token2", []string{"token1", "token2"}},
		{"empty editors", "", []string{""}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Collection{Editors: tt.editors}
			got := c.GetEditors()
			if len(got) != len(tt.want) {
				t.Fatalf("GetEditors() = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("GetEditors() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestCollectionIsEditor(t *testing.T) {
	c := Collection{Editors: "token1, token2"}

	if !c.IsEditor("token1") {
		t.Fatal("expected token1 to be an editor")
	}
	if !c.IsEditor("token2") {
		t.Fatal("expected token2 to be an editor")
	}
	if c.IsEditor("token3") {
		t.Fatal("did not expect token3 to be an editor")
	}
	if c.IsEditor("") {
		t.Fatal("did not expect empty token to be an editor")
	}
}
