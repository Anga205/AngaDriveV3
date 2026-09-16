package runners

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
)

// removeExtension strips the final extension (everything after the last dot)
// from a filename. If there is no dot, the filename is returned unchanged.
func removeExtension(filename string) string {
	for i := len(filename) - 1; i >= 0; i-- {
		if filename[i] == '.' {
			return filename[:i]
		}
	}
	return filename
}

// sha256sum returns the lowercase hex SHA-256 digest of the file at filepath.
func sha256sum(filepath string) (string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
