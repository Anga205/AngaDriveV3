package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting current working directory:", err)
		return
	}

	distPath := cwd + "/dist"
	if _, err := os.Stat(distPath); os.IsNotExist(err) {
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
	} else {
		fmt.Println("Found existing dist directory, skipping build process.")
	}

	routes := []string{"/", "/my_drive"}

	r := gin.Default()

	for _, route := range routes {
		r.GET(route, func(c *gin.Context) {
			c.File(distPath + "/index.html")
		})
	}

	err = filepath.Walk(distPath, func(path string, info os.FileInfo, err error) error {
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
		return
	}

	r.Run()
}
