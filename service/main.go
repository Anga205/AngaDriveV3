package main

import (
	"service/global"
	"service/setup"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	err := setup.SetupFrontend(r)
	if err != nil {
		panic(err)
	}

	setup.SetupDB(global.UploadedFilesDir)

	setup.SetupWebsocket(r)

	r.Run()
}
