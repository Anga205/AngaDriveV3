package endpoints

import (
	"angadrive/socketHandler"
	"angadrive/vars"
	"bytes"
	"compress/gzip"
	"fmt"
	"mime"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	brotli "github.com/andybalholm/brotli"
	"github.com/gin-gonic/gin"
)

type CachedFile struct {
	Raw         []byte
	Gzip        []byte
	Brotli      []byte
	ContentType string
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
	buildCmd.Env = append(os.Environ(), "VITE_ASSETS_URL="+vars.AssetsURL)
	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("[compileFrontend] error running 'bun run build': %w", err)
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

func setupRoutes(r *gin.Engine, cache map[string]CachedFile) {
	indexFile, ok := cache["/index.html"]
	if !ok {
		fmt.Println("Error caching files: missing /index.html")
		return
	}

	routes := []string{"/", "/my_drive", "/my_collections", "/collection", "/account"}
	for _, route := range routes {
		route := route
		r.GET(route, func(c *gin.Context) {
			if c.Request.Host == vars.WebURL {
				go socketHandler.SiteActivityPulse()
				serveCachedFile(c, indexFile)
			} else if route == "/" && c.Request.Host == vars.AssetsURL {
				scheme := "http"
				if c.Request.TLS != nil {
					scheme = "https"
				} else if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
					scheme = proto
				}
				c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("%s://%s/", scheme, vars.WebURL))
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
				if c.Request.Host == vars.WebURL {
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
	if _, err := os.Stat(distPath); os.IsNotExist(err) {
		fmt.Println("[GIN-debug] Building frontend...")
		err := compileFrontend()
		if err != nil {
			return fmt.Errorf("[SetupFrontend] error compiling frontend: %w", err)
		}
	} else {
		fmt.Println("[GIN-debug] Found existing dist directory, skipping build process.")
	}
	cache, err := loadFrontendCache(distPath)
	if err != nil {
		return fmt.Errorf("[SetupFrontend] error loading frontend cache: %w", err)
	}
	setupRoutes(r, cache)
	fmt.Println("[GIN-debug] Frontend setup completed.")
	return nil
}
