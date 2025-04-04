package main

import (
	"service/global"
	"service/setup"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	setup.SetupFrontend(r)
	setup.SetupDB(global.UploadedFilesDir)

	setup.SetupWebsocket(r)

	r.Run()
}
