package endpoints

import (
	"angadrive/globals"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// hashFrontendSource computes a deterministic hash over the frontend source
// files that influence the built output. It walks the web directory (excluding
// node_modules, dist, and the hash file itself) and hashes each file's relative
// path and contents. The result is used to detect whether an existing dist
// build is stale.
func hashFrontendSource() (string, error) {
	hasher := sha256.New()

	webDir := "web"
	err := filepath.Walk(webDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Skip directories that don't affect the build output.
			if info.Name() == "node_modules" || info.Name() == "dist" {
				return filepath.SkipDir
			}
			return nil
		}

		relPath, err := filepath.Rel(webDir, path)
		if err != nil {
			return err
		}
		relPath = filepath.ToSlash(relPath)

		// Ignore the hash file itself so it never invalidates the build.
		if relPath == frontendHashFile {
			return nil
		}

		// Hash the relative path so renames/moves invalidate the build.
		if _, err := io.WriteString(hasher, relPath); err != nil {
			return err
		}
		if _, err := io.WriteString(hasher, "\x00"); err != nil {
			return err
		}

		// Hash the file contents.
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		if _, err := io.Copy(hasher, file); err != nil {
			return err
		}
		if _, err := io.WriteString(hasher, "\x00"); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func compileFrontend() error {
	fmt.Println("[GIN-debug] Building the dist directory...")
	if _, err := os.Stat(filepath.Join("web", "node_modules")); os.IsNotExist(err) {
		fmt.Println("node_modules not found, running 'bun install'...")
		installCmd := exec.Command("bun", "install")
		installCmd.Dir = "web"
		if err := installCmd.Run(); err != nil {
			return fmt.Errorf("[compileFrontend] error running 'bun install': %w", err)
		}
	}
	buildCmd := exec.Command("bun", "run", "build")
	buildCmd.Dir = "web"
	buildCmd.Env = append(os.Environ(), "VITE_ASSETS_URL="+globals.AssetsURL)
	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("[compileFrontend] error running 'bun run build': %w", err)
	}
	// Remove any stale dist directory left over from a previous build before
	// moving the freshly built one into place.
	if err := os.RemoveAll("./dist"); err != nil {
		return fmt.Errorf("[compileFrontend] error removing stale dist directory: %w", err)
	}
	if err := os.Rename("web/dist", "./dist"); err != nil {
		return fmt.Errorf("[compileFrontend] error moving dist directory: %w", err)
	}
	fmt.Println("[GIN-debug] Build completed and dist directory moved to backend.")
	return nil
}

func SetupFrontend(r *gin.Engine) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("[SetupFrontend] error getting current working directory: %w", err)
	}
	distPath := filepath.Join(cwd, "dist")

	// Compute the hash of the current frontend source. This is used to decide
	// whether an existing dist build is still up to date.
	sourceHash, err := hashFrontendSource()
	if err != nil {
		return fmt.Errorf("[SetupFrontend] error hashing frontend source: %w", err)
	}

	distUpToDate := false
	if hashBytes, err := os.ReadFile(filepath.Join(distPath, frontendHashFile)); err == nil {
		distUpToDate = strings.TrimSpace(string(hashBytes)) == sourceHash
	}

	if !distUpToDate {
		fmt.Println("[GIN-debug] Frontend source changed or dist missing, rebuilding...")
		err := compileFrontend()
		if err != nil {
			return fmt.Errorf("[SetupFrontend] error compiling frontend: %w", err)
		}
		// Record the source hash alongside the freshly built dist so future
		// startups can detect whether the build is still up to date.
		if err := os.WriteFile(filepath.Join(distPath, frontendHashFile), []byte(sourceHash), 0o644); err != nil {
			return fmt.Errorf("[SetupFrontend] error writing frontend hash file: %w", err)
		}
	} else {
		fmt.Println("[GIN-debug] Found up-to-date dist directory, skipping build process.")
	}

	cache, err := loadFrontendCache(distPath)
	if err != nil {
		return fmt.Errorf("[SetupFrontend] error loading frontend cache: %w", err)
	}
	setupRoutes(r, cache)
	fmt.Println("[GIN-debug] Frontend setup completed.")
	return nil
}
