package endpoints

import (
	"angadrive/globals"
	"angadrive/requestHandler"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	brotli "github.com/andybalholm/brotli"
	"github.com/gin-gonic/gin"
)

// frontendHashFile is the name of the file stored inside the dist directory
// that records the hash of the frontend source used to produce that build.
const frontendHashFile = ".frontend-hash"

type CachedFile struct {
	Raw         []byte
	Gzip        []byte
	Brotli      []byte
	ContentType string
}

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

type acceptEncodingPreference struct {
	quality float64
	present bool
}

func buildCachedFile(path string, raw []byte) CachedFile {
	contentType := detectContentType(path)
	cachedFile := CachedFile{
		Raw:         raw,
		ContentType: contentType,
	}

	if isCompressible(contentType) {
		cachedFile.Gzip = compressGzip(raw)
		cachedFile.Brotli = compressBrotli(raw)
	}

	return cachedFile
}

func compressGzip(raw []byte) []byte {
	var buffer bytes.Buffer
	writer, err := gzip.NewWriterLevel(&buffer, gzip.BestCompression)
	if err != nil {
		return nil
	}
	if _, err := writer.Write(raw); err != nil {
		_ = writer.Close()
		return nil
	}
	if err := writer.Close(); err != nil {
		return nil
	}
	return buffer.Bytes()
}

func compressBrotli(raw []byte) []byte {
	var buffer bytes.Buffer
	writer := brotli.NewWriterLevel(&buffer, 11)
	if _, err := writer.Write(raw); err != nil {
		_ = writer.Close()
		return nil
	}
	if err := writer.Close(); err != nil {
		return nil
	}
	return buffer.Bytes()
}

func detectContentType(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".html", ".htm":
		return "text/html; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".js", ".mjs":
		return "application/javascript"
	case ".json", ".webmanifest":
		return "application/json"
	case ".svg":
		return "image/svg+xml"
	case ".xml":
		return "application/xml"
	case ".wasm":
		return "application/wasm"
	}

	if contentType := mime.TypeByExtension(filepath.Ext(path)); contentType != "" {
		return contentType
	}

	return "application/octet-stream"
}

func isCompressible(contentType string) bool {
	normalized := strings.ToLower(contentType)
	switch {
	case strings.HasPrefix(normalized, "text/"):
		return true
	case strings.Contains(normalized, "javascript"):
		return true
	case strings.Contains(normalized, "json"):
		return true
	case strings.Contains(normalized, "xml"):
		return true
	case strings.Contains(normalized, "svg+xml"):
		return true
	case normalized == "application/wasm":
		return true
	default:
		return false
	}
}

func parseAcceptEncoding(header string) map[string]acceptEncodingPreference {
	preferences := make(map[string]acceptEncodingPreference)
	if header == "" {
		return preferences
	}

	for _, rawItem := range strings.Split(header, ",") {
		item := strings.TrimSpace(rawItem)
		if item == "" {
			continue
		}

		parts := strings.Split(item, ";")
		name := strings.ToLower(strings.TrimSpace(parts[0]))
		if name == "" {
			continue
		}

		quality := 1.0
		for _, rawParam := range parts[1:] {
			param := strings.TrimSpace(rawParam)
			if len(param) < 2 || !strings.EqualFold(param[:2], "q=") {
				continue
			}

			parsedQuality, err := strconv.ParseFloat(strings.TrimSpace(param[2:]), 64)
			if err != nil {
				quality = 0
			} else {
				quality = parsedQuality
			}
			break
		}

		if current, ok := preferences[name]; !ok || quality > current.quality {
			preferences[name] = acceptEncodingPreference{quality: quality, present: true}
		}
	}

	return preferences
}

func encodingQuality(preferences map[string]acceptEncodingPreference, encoding string) (float64, bool) {
	if preference, ok := preferences[encoding]; ok {
		return preference.quality, true
	}
	if preference, ok := preferences["*"]; ok {
		return preference.quality, true
	}
	return 0, false
}

func negotiateEncoding(acceptEncoding string, cachedFile CachedFile) (string, []byte) {
	if len(cachedFile.Gzip) == 0 && len(cachedFile.Brotli) == 0 {
		return "", cachedFile.Raw
	}

	preferences := parseAcceptEncoding(acceptEncoding)
	brQuality, brAccepted := encodingQuality(preferences, "br")
	gzipQuality, gzipAccepted := encodingQuality(preferences, "gzip")

	if brAccepted && brQuality > 0 && len(cachedFile.Brotli) > 0 {
		if !gzipAccepted || brQuality >= gzipQuality || gzipQuality <= 0 {
			return "br", cachedFile.Brotli
		}
	}

	if gzipAccepted && gzipQuality > 0 && len(cachedFile.Gzip) > 0 {
		return "gzip", cachedFile.Gzip
	}

	if brAccepted && brQuality > 0 && len(cachedFile.Brotli) > 0 {
		return "br", cachedFile.Brotli
	}

	return "", cachedFile.Raw
}

func loadFrontendCache(distPath string) (map[string]CachedFile, error) {
	cache := make(map[string]CachedFile)

	err := filepath.Walk(distPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			relativePath, err := filepath.Rel(distPath, path)
			if err != nil {
				return err
			}
			relativePath = "/" + filepath.ToSlash(relativePath)
			// Skip the internal hash marker file; it is not part of the SPA.
			if relativePath == "/"+frontendHashFile {
				return nil
			}
			cache[relativePath] = buildCachedFile(relativePath, raw)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return cache, nil
}

func serveCachedFile(c *gin.Context, cachedFile CachedFile) {
	encoding, body := negotiateEncoding(c.GetHeader("Accept-Encoding"), cachedFile)
	headers := c.Writer.Header()
	headers.Set("Vary", "Accept-Encoding")
	headers.Set("Content-Type", cachedFile.ContentType)
	if encoding != "" {
		headers.Set("Content-Encoding", encoding)
	}

	c.Data(http.StatusOK, cachedFile.ContentType, body)
}

// collectionHandler serves the SPA for the /collection and /collection/* paths.
// If a legacy 'id' query parameter is present it redirects with a permanent 301
// to the canonical path-based collection URL.
func collectionHandler(indexFile CachedFile) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Host != globals.WebURL {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		legacyID := c.Query("id")
		if legacyID != "" {
			// Split the space-separated navigation path into individual IDs.
			// c.Query returns the decoded value, so "+" and %20 already became
			// spaces and we use strings.Fields (which also drops empties).
			ids := strings.Fields(legacyID)
			if len(ids) > 0 {
				escaped := make([]string, 0, len(ids))
				for _, id := range ids {
					escaped = append(escaped, url.PathEscape(id))
				}
				c.Redirect(http.StatusMovedPermanently, "/collection/"+strings.Join(escaped, "/"))
				return
			}
		}

		// No legacy id (e.g. /collection or /collection/) -> serve the SPA.
		go requestHandler.SiteActivityPulse()
		serveCachedFile(c, indexFile)
	}
}

func setupRoutes(r *gin.Engine, cache map[string]CachedFile) {
	indexFile, ok := cache["/index.html"]
	if !ok {
		fmt.Println("Error caching files: missing /index.html")
		return
	}

	// Collection routes: /collection and /collection/*path. The catch-all also
	// covers /collection/ (empty trailing segment via *path == "/"). Both use
	// the same handler so a legacy ?id= query on /collection or /collection/
	// produces an HTTP 301 redirect to the canonical path-based URL.
	r.GET("/collection", collectionHandler(indexFile))
	r.GET("/collection/*path", collectionHandler(indexFile))

	routes := []string{"/", "/my_drive", "/my_collections", "/account"}
	for _, route := range routes {
		route := route
		r.GET(route, func(c *gin.Context) {
			if c.Request.Host == globals.WebURL {
				go requestHandler.SiteActivityPulse()
				serveCachedFile(c, indexFile)
			} else if route == "/" && c.Request.Host == globals.AssetsURL {
				scheme := "http"
				if c.Request.TLS != nil {
					scheme = "https"
				} else if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
					scheme = proto
				}
				c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("%s://%s/", scheme, globals.WebURL))
			} else {
				c.AbortWithStatus(http.StatusNotFound)
			}
		})
	}

	for relPath, cachedFile := range cache { // register all files in the dist directory
		relPath := relPath
		cachedFile := cachedFile
		if relPath != "/index.html" { // since index.html is handled separately, we dont want to register it again
			r.GET(relPath, func(c *gin.Context) {
				if c.Request.Host == globals.WebURL {
					serveCachedFile(c, cachedFile)
				} else {
					c.AbortWithStatus(http.StatusNotFound)
				}
			})
		}
	}
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
