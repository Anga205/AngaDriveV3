package setup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

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

func setupRoutes(r *gin.Engine, distPath string) {
	r.GET("/", func(c *gin.Context) {
		c.File(distPath + "/index.html")
	})
	r.GET("/my_drive", func(c *gin.Context) {
		c.File(distPath + "/index.html")
	})

	err := filepath.Walk(distPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			relativePath := path[len(distPath):]
			r.GET(relativePath, func(c *gin.Context) {
				c.File(path)
			})
		}
		return nil
	})
	if err != nil {
		fmt.Println("Error walking through dist directory:", err)
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
