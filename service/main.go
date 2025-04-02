package main

import (
	"service/setup"

	"github.com/gin-gonic/gin"
)

func main() {

	uploadedFilesDir := "uploaded_files"

	r := gin.Default()
	setup.SetupFrontend(r)
	setup.SetupDB(uploadedFilesDir)
	r.Run()
}
