package database

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

// TestSha256OfFile verifies that sha256OfFile produces the expected SHA-256
// digest for a file's bytes.
func TestSha256OfFile(t *testing.T) {
	resetState(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "sample.txt")
	content := []byte("AngaDrive SHA-256 migration test content")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	expected := sha256.New()
	expected.Write(content)
	want := hex.EncodeToString(expected.Sum(nil))

	got, err := sha256OfFile(path)
	if err != nil {
		t.Fatalf("sha256OfFile failed: %v", err)
	}
	if got != want {
		t.Fatalf("sha256OfFile = %q, want %q", got, want)
	}
}

// TestSha256DifferentFiles verifies that different files produce different
// SHA-256 digests.
func TestSha256DifferentFiles(t *testing.T) {
	resetState(t)

	dir := t.TempDir()
	pathA := filepath.Join(dir, "a.txt")
	pathB := filepath.Join(dir, "b.txt")
	if err := os.WriteFile(pathA, []byte("content A"), 0644); err != nil {
		t.Fatalf("write a: %v", err)
	}
	if err := os.WriteFile(pathB, []byte("content B"), 0644); err != nil {
		t.Fatalf("write b: %v", err)
	}

	hashA, err := sha256OfFile(pathA)
	if err != nil {
		t.Fatalf("sha256OfFile(a): %v", err)
	}
	hashB, err := sha256OfFile(pathB)
	if err != nil {
		t.Fatalf("sha256OfFile(b): %v", err)
	}
	if hashA == hashB {
		t.Fatal("different files should produce different SHA-256 digests")
	}
}

// TestSha256IdenticalFilesShareHash verifies that two files with identical
// content produce the same SHA-256 digest (the basis for physical sharing).
func TestSha256IdenticalFilesShareHash(t *testing.T) {
	resetState(t)

	dir := t.TempDir()
	pathA := filepath.Join(dir, "a.txt")
	pathB := filepath.Join(dir, "b.txt")
	content := []byte("identical content")
	if err := os.WriteFile(pathA, content, 0644); err != nil {
		t.Fatalf("write a: %v", err)
	}
	if err := os.WriteFile(pathB, content, 0644); err != nil {
		t.Fatalf("write b: %v", err)
	}

	hashA, err := sha256OfFile(pathA)
	if err != nil {
		t.Fatalf("sha256OfFile(a): %v", err)
	}
	hashB, err := sha256OfFile(pathB)
	if err != nil {
		t.Fatalf("sha256OfFile(b): %v", err)
	}
	if hashA != hashB {
		t.Fatal("identical files should produce the same SHA-256 digest")
	}
}

// TestCheckForFilesWithSha256sum verifies the reference-counting query used
// during deletion: a hash is "still referenced" while any FileData row (in
// cache or database) holds it.
func TestCheckForFilesWithSha256sum(t *testing.T) {
	resetState(t)
	forceLoad(t)

	// No rows yet -> not referenced.
	if CheckForFilesWithSha256sum("sha256-nonexistent") {
		t.Fatal("expected false for unknown hash")
	}

	f1 := insertTestFile(t, "fileA") // Sha256sum = "sha256-fileA"
	if !CheckForFilesWithSha256sum(f1.Sha256sum) {
		t.Fatal("expected true for hash present in cache")
	}

	// A second FileData referencing the same hash keeps it referenced.
	f2 := FileData{
		OriginalFileName: "copy.txt",
		FileDirectory:    "fileB",
		AccountToken:     "anon-token",
		FileSize:         100,
		Timestamp:        2,
		Sha256sum:        f1.Sha256sum,
	}
	if err := f2.Insert(); err != nil {
		t.Fatalf("insert f2: %v", err)
	}
	if !CheckForFilesWithSha256sum(f1.Sha256sum) {
		t.Fatal("expected true while two rows share the hash")
	}

	// Delete one row; the other still references the hash.
	if err := DeleteFile(f1, func(Collection) {}); err != nil {
		t.Fatalf("delete f1: %v", err)
	}
	if !CheckForFilesWithSha256sum(f1.Sha256sum) {
		t.Fatal("expected true while one row still references the hash")
	}

	// Delete the last row; the hash is no longer referenced.
	if err := DeleteFile(f2, func(Collection) {}); err != nil {
		t.Fatalf("delete f2: %v", err)
	}
	if CheckForFilesWithSha256sum(f1.Sha256sum) {
		t.Fatal("expected false after the final reference is removed")
	}
}

// TestSha256MigrationFromMd5 verifies that migrateMd5ToSha256 safely converts
// a legacy MD5-named physical file and its database row to SHA-256 naming.
func TestSha256MigrationFromMd5(t *testing.T) {
	// Build a dedicated database directory so we control both the DB and the
	// physical files, simulating a pre-existing legacy installation.
	dir := t.TempDir()
	os.Setenv("SAVE_DRIVE_RAM", "true")
	defer os.Unsetenv("SAVE_DRIVE_RAM")
	if err := InitializeDatabase(); err != nil {
		t.Fatalf("InitializeDatabase failed: %v", err)
	}

	// Simulate a legacy schema by adding the old md5sum column back.
	if err := GetDB().Exec("ALTER TABLE file_data ADD COLUMN md5sum TEXT").Error; err != nil {
		t.Fatalf("add legacy md5sum column: %v", err)
	}

	iDir := filepath.Join(dir, "i")
	if err := os.MkdirAll(iDir, os.ModePerm); err != nil {
		t.Fatalf("mkdir i: %v", err)
	}

	content := []byte("legacy file content for migration")
	// Compute the SHA-256 we expect after migration.
	expectedHash := sha256.New()
	expectedHash.Write(content)
	wantSha := hex.EncodeToString(expectedHash.Sum(nil))

	// Use a fake MD5 name (32 hex chars) for the physical file.
	legacyMd5 := "0123456789abcdef0123456789abcdef"
	legacyPath := filepath.Join(iDir, legacyMd5+".txt")
	if err := os.WriteFile(legacyPath, content, 0644); err != nil {
		t.Fatalf("write legacy file: %v", err)
	}

	// Insert a row with the legacy md5sum column set directly via raw SQL.
	// The FileData model no longer has Md5sum, so we use the raw column.
	if err := GetDB().Exec(
		"INSERT INTO file_data (original_file_name, file_directory, account_token, file_size, timestamp, md5sum) VALUES (?, ?, ?, ?, ?, ?)",
		"legacy.txt", "legacyDir", "anon-token", int64(len(content)), int64(1), legacyMd5+".txt",
	).Error; err != nil {
		t.Fatalf("insert legacy row: %v", err)
	}

	// Run the migration.
	if err := migrateMd5ToSha256(dir); err != nil {
		t.Fatalf("migrateMd5ToSha256 failed: %v", err)
	}

	// The physical file should now be named by SHA-256.
	newPath := filepath.Join(iDir, wantSha+".txt")
	if _, err := os.Stat(newPath); os.IsNotExist(err) {
		t.Fatalf("expected SHA-256-named physical file at %s", newPath)
	}
	if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
		t.Fatal("legacy MD5-named physical file should have been renamed away")
	}

	// The row should now have sha256sum set and md5sum dropped.
	var row FileData
	if err := GetDB().Where("file_directory = ?", "legacyDir").First(&row).Error; err != nil {
		t.Fatalf("fetch migrated row: %v", err)
	}
	if row.Sha256sum != wantSha+".txt" {
		t.Fatalf("row.Sha256sum = %q, want %q", row.Sha256sum, wantSha+".txt")
	}
}

// TestSha256MigrationDeduplicatedRows verifies that when multiple FileData rows
// share the same MD5 physical file, the migration renames the physical file
// once and backfills both rows with the same SHA-256 value.
func TestSha256MigrationDeduplicatedRows(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("SAVE_DRIVE_RAM", "true")
	defer os.Unsetenv("SAVE_DRIVE_RAM")
	if err := InitializeDatabase(); err != nil {
		t.Fatalf("InitializeDatabase failed: %v", err)
	}

	// Simulate a legacy schema by adding the old md5sum column back.
	if err := GetDB().Exec("ALTER TABLE file_data ADD COLUMN md5sum TEXT").Error; err != nil {
		t.Fatalf("add legacy md5sum column: %v", err)
	}

	iDir := filepath.Join(dir, "i")
	if err := os.MkdirAll(iDir, os.ModePerm); err != nil {
		t.Fatalf("mkdir i: %v", err)
	}

	content := []byte("shared deduplicated content")
	expectedHash := sha256.New()
	expectedHash.Write(content)
	wantSha := hex.EncodeToString(expectedHash.Sum(nil))

	legacyMd5 := "abcdefabcdefabcdefabcdefabcdefab"
	legacyPath := filepath.Join(iDir, legacyMd5+".txt")
	if err := os.WriteFile(legacyPath, content, 0644); err != nil {
		t.Fatalf("write legacy file: %v", err)
	}

	// Two logical files referencing the same physical MD5 file.
	for _, dirName := range []string{"legacyA", "legacyB"} {
		if err := GetDB().Exec(
			"INSERT INTO file_data (original_file_name, file_directory, account_token, file_size, timestamp, md5sum) VALUES (?, ?, ?, ?, ?, ?)",
			"shared.txt", dirName, "anon-token", int64(len(content)), int64(1), legacyMd5+".txt",
		).Error; err != nil {
			t.Fatalf("insert legacy row %s: %v", dirName, err)
		}
	}

	if err := migrateMd5ToSha256(dir); err != nil {
		t.Fatalf("migrateMd5ToSha256 failed: %v", err)
	}

	// The physical file should be renamed once to the SHA-256 name.
	newPath := filepath.Join(iDir, wantSha+".txt")
	if _, err := os.Stat(newPath); os.IsNotExist(err) {
		t.Fatalf("expected SHA-256-named physical file at %s", newPath)
	}
	if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
		t.Fatal("legacy MD5-named physical file should have been renamed away")
	}

	// Both rows should reference the same SHA-256 value.
	for _, dirName := range []string{"legacyA", "legacyB"} {
		var row FileData
		if err := GetDB().Where("file_directory = ?", dirName).First(&row).Error; err != nil {
			t.Fatalf("fetch migrated row %s: %v", dirName, err)
		}
		if row.Sha256sum != wantSha+".txt" {
			t.Fatalf("row %s Sha256sum = %q, want %q", dirName, row.Sha256sum, wantSha+".txt")
		}
	}
}
