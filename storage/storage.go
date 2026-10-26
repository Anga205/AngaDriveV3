// Package storage centralizes filesystem layout and file-level helpers
// (paths, hashing, filename generation). It has no dependency on the database
// or transport layers so it can be reused and tested in isolation.
package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"angadrive/config"
)

// Sub-directory names under the upload root.
const (
	FilesDirName         = "i"
	ChunksDirName        = "tmp_chunks"
	PDFPreviewsDirName   = "pdf_previews"
	ImagePreviewsDirName = "image_previews"
	VideoPreviewsDirName = "video_previews"
	ReposDirName         = "repos"
)

// Root returns the configured upload root directory.
func Root() string { return config.UploadDir }

// FilesDir returns the directory holding content-addressed original files.
func FilesDir() string { return filepath.Join(Root(), FilesDirName) }

// ChunksDir returns the directory holding in-progress upload chunks.
func ChunksDir() string { return filepath.Join(Root(), ChunksDirName) }

// PDFPreviewsDir returns the directory holding generated PDF previews.
func PDFPreviewsDir() string { return filepath.Join(Root(), PDFPreviewsDirName) }

// ImagePreviewsDir returns the directory holding generated image previews.
func ImagePreviewsDir() string { return filepath.Join(Root(), ImagePreviewsDirName) }

// VideoPreviewsDir returns the directory holding generated video previews.
func VideoPreviewsDir() string { return filepath.Join(Root(), VideoPreviewsDirName) }

// ReposDir returns the directory used for cloned repositories.
func ReposDir() string { return filepath.Join(Root(), ReposDirName) }

// FilePath returns the on-disk path for a stored file identified by its
// content-addressed name (sha256, optionally with an extension).
func FilePath(name string) string { return filepath.Join(FilesDir(), name) }

// PDFPreviewPath returns the on-disk path for a PDF preview of the given hash.
func PDFPreviewPath(hash string) string { return filepath.Join(PDFPreviewsDir(), hash+".jpg") }

// ImagePreviewPath returns the on-disk path for an image preview of the given hash.
func ImagePreviewPath(hash string) string { return filepath.Join(ImagePreviewsDir(), hash) }

// VideoPreviewPath returns the on-disk path for a video preview of the given hash.
func VideoPreviewPath(hash string) string { return filepath.Join(VideoPreviewsDir(), hash+".webp") }

// EnsureLayout creates the upload root and all known sub-directories.
func EnsureLayout() error {
	dirs := []string{
		Root(),
		FilesDir(),
		ChunksDir(),
		PDFPreviewsDir(),
		ImagePreviewsDir(),
		VideoPreviewsDir(),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("storage: create %s: %w", dir, err)
		}
	}
	return nil
}

// SHA256File returns the lowercase hex SHA-256 digest of the file at path.
func SHA256File(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// Extension returns the extension of filename without the leading dot.
func Extension(filename string) string {
	return strings.TrimPrefix(filepath.Ext(filename), ".")
}

// RemoveExtension strips the final extension from a filename.
func RemoveExtension(filename string) string {
	return strings.TrimSuffix(filename, filepath.Ext(filename))
}

const allowedFilenameCharacters = "qwertyuiopasdfghjklzxcvbnm1234567890"

// UniqueFileName generates a random 12-character name (preserving the original
// extension) that does not already exist according to exists. The exists
// predicate lets callers supply their own uniqueness check (e.g. a DB lookup)
// without storage depending on the database.
func UniqueFileName(filename string, exists func(string) bool) string {
	for {
		name := randomName(filename)
		if exists == nil || !exists(name) {
			return name
		}
	}
}

func randomName(filename string) string {
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	var sb strings.Builder
	for i := 0; i < 12; i++ {
		sb.WriteByte(allowedFilenameCharacters[random.Intn(len(allowedFilenameCharacters))])
	}
	if ext := filepath.Ext(filename); ext != "" {
		sb.WriteString(ext)
	}
	return sb.String()
}
