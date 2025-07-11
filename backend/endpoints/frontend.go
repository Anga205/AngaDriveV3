package endpoints

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"service/socketHandler"
	"service/vars"
	"strings"

	"github.com/gin-gonic/gin"
)

func compileFrontend() error {
	fmt.Println("[GIN-debug] Building the dist directory...")
	if err := os.Chdir(".."); err != nil {
		return fmt.Errorf("[compileFrontend] error changing to parent directory: %w", err)
	}
	if err := os.Chdir("frontend"); err != nil {
		return fmt.Errorf("[compileFrontend] error changing to 'frontend' directory: %w", err)
	}
	if _, err := os.Stat("node_modules"); os.IsNotExist(err) {
		fmt.Println("node_modules not found, running 'bun install'...")
		if err := exec.Command("bun", "install").Run(); err != nil {
			return fmt.Errorf("[compileFrontend] error running 'bun install': %w", err)
		}
	}
	if err := exec.Command("bun", "run", "build").Run(); err != nil {
		return fmt.Errorf("[compileFrontend] error running 'bun run build': %w", err)
	}
	if err := os.Chdir(".."); err != nil {
		return fmt.Errorf("[compileFrontend] error changing to parent directory: %w", err)
	}
	if err := os.Rename("frontend/dist", "backend/dist"); err != nil {
		return fmt.Errorf("[compileFrontend] error moving dist directory: %w", err)
	}
	if err := os.Chdir("backend"); err != nil {
		return fmt.Errorf("[compileFrontend] error changing to 'backend' directory: %w", err)
	}
	fmt.Println("[GIN-debug] Build completed and dist directory moved to backend.")
	return nil
}

var fileCache = make(map[string][]byte)

func setupRoutes(r *gin.Engine, distPath string) {
	err := filepath.Walk(distPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			relativePath := path[len(distPath):]
			fileCache[relativePath] = data
		}
		return nil
	})
	if err != nil {
		fmt.Println("Error caching files:", err)
		return
	}

	r.GET("/", func(c *gin.Context) {
		fmt.Println(c.Request.Host)
		if c.Request.Host == vars.FrontendURL {
			go socketHandler.SiteActivityPulse()
			c.Data(http.StatusOK, "text/html", fileCache["/index.html"])
		} else {
			c.AbortWithStatus(http.StatusNotFound)
		}
	})

	r.GET("/my_drive", func(c *gin.Context) {
		if c.Request.Host == vars.FrontendURL {
			go socketHandler.SiteActivityPulse()
			c.Data(http.StatusOK, "text/html", fileCache["/index.html"])
		} else {
			c.AbortWithStatus(http.StatusNotFound)
		}
	})

	r.GET("/my_collections", func(c *gin.Context) {
		if c.Request.Host == vars.FrontendURL {
			go socketHandler.SiteActivityPulse()
			c.Data(http.StatusOK, "text/html", fileCache["/index.html"])
		} else {
			c.AbortWithStatus(http.StatusNotFound)
		}
	})

	r.GET("/collection", func(c *gin.Context) {
		if c.Request.Host == vars.FrontendURL {
			go socketHandler.SiteActivityPulse()
			c.Data(http.StatusOK, "text/html", fileCache["/index.html"])
		} else {
			c.AbortWithStatus(http.StatusNotFound)
		}
	})

	r.GET("/account", func(c *gin.Context) {
		if c.Request.Host == vars.FrontendURL {
			go socketHandler.SiteActivityPulse()
			c.Data(http.StatusOK, "text/html", fileCache["/index.html"])
		} else {
			c.AbortWithStatus(http.StatusNotFound)
		}
	})

	for relPath := range fileCache { // register all files in the dist directory
		if relPath != "/index.html" { // since index.html is handled separately, we dont want to register it again
			r.GET(relPath, func(c *gin.Context) {
				if c.Request.Host == vars.FrontendURL {
					c.Data(
						http.StatusOK,
						getContentType(relPath),
						fileCache[relPath],
					)
				} else {
					c.AbortWithStatus(http.StatusNotFound)
				}
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

func SetupFrontend(r *gin.Engine) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("[SetupFrontend] error getting current working directory: %w", err)
	}
	if _, err := os.Stat(cwd + "/dist"); os.IsNotExist(err) {
		fmt.Println("[GIN-debug] Building frontend...")
		err := compileFrontend()
		if err != nil {
			return fmt.Errorf("[SetupFrontend] error compiling frontend: %w", err)
		}
	} else {
		fmt.Println("[GIN-debug] Found existing dist directory, skipping build process.")
	}
	distPath := cwd + "/dist"
	setupRoutes(r, distPath)
	fmt.Println("[GIN-debug] Frontend setup completed.")
	return nil
}
