package setup

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

func compileFrontend() {
	fmt.Println("Building the dist directory...")
	if err := os.Chdir(".."); err != nil {
		fmt.Println("Error changing directory:", err)
		return
	}
	if err := os.Chdir("web"); err != nil {
		fmt.Println("Error changing to 'web' directory:", err)
		return
	}
	if _, err := os.Stat("node_modules"); os.IsNotExist(err) {
		fmt.Println("node_modules not found, running 'bun install'...")
		if err := exec.Command("bun", "install").Run(); err != nil {
			fmt.Println("Error running 'bun install':", err)
			return
		}
	}
	if err := exec.Command("bun", "run", "build").Run(); err != nil {
		fmt.Println("Error running 'bun run build':", err)
		return
	}
	if err := os.Chdir(".."); err != nil {
		fmt.Println("Error changing back to parent directory:", err)
		return
	}
	if err := os.Rename("web/dist", "service/dist"); err != nil {
		fmt.Println("Error moving 'web/dist' to 'service/dist':", err)
		return
	}
	fmt.Println("Build completed and dist directory moved to service.")
}

var fileCache = make(map[string][]byte) // Map where the file paths are keys and the file contents are values

// im doing hella vibe-coding rn, dont mind the comments, its just to help me understand later
func setupRoutes(r *gin.Engine, distPath string) {
	// Recursively walk through the distPath directory
	err := filepath.Walk(distPath, func(path string, info os.FileInfo, err error) error { // disposable function to run for each file
		if err != nil {
			return err // Handle error, idk what errors to expect, so just return for now
		}
		if !info.IsDir() {
			data, err := os.ReadFile(path) // Read file
			if err != nil {
				return err
			}
			relativePath := path[len(distPath):] // path is the path to the file, distPath is the path to the dist directory
			// ^^^^  we extract the relative path by removing the distPath prefix
			fileCache[relativePath] = data // Cache the content by adding it to the map
		}
		return nil // does this need to be here? it doesnt do anything but golang keeps throwing errors if i remove it
	})
	if err != nil {
		fmt.Println("Error caching files:", err)
		return
	}

	// Define routes, since im handling the routing on the client side, everything here just points to the same index.html file
	r.GET("/", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html", fileCache["/index.html"])
	})

	r.GET("/my_drive", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html", fileCache["/index.html"])
	})

	// Register routes for all other files (from cache)
	for relPath := range fileCache {
		if relPath != "/index.html" { // Skip already registered paths
			r.GET(relPath, func(c *gin.Context) {
				c.Data(
					http.StatusOK, // this just means 200, i couldve just put 200 here but makes it look like i know what im doing
					getContentType(relPath),
					fileCache[relPath],
				)
			})
		}
	}
}

func getContentType(path string) string {
	switch {
	case strings.HasSuffix(path, ".css"):
		return "text/css"
	case strings.HasSuffix(path, ".js"):
		return "application/javascript"
	case strings.HasSuffix(path, ".png"): // i dont even have any png files but whatever
		return "image/png"
	case strings.HasSuffix(path, ".svg"):
		return "image/svg+xml"
	default:
		return "text/html"
	}
}

func SetupFrontend(r *gin.Engine) {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting current working directory:", err)
		return
	}
	if _, err := os.Stat(cwd + "/dist"); os.IsNotExist(err) {
		fmt.Println("Building frontend...")
		compileFrontend()
	} else {
		fmt.Println("Found existing dist directory, skipping build process.")
	}
	distPath := cwd + "/dist"
	setupRoutes(r, distPath)
	fmt.Println("Frontend setup completed.")
}
