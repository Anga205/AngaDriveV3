package main

import (
	"service/database"
	"service/setup"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	err := setup.SetupFrontend(r)
	if err != nil {
		panic(err)
	}
	database.InitializeDatabase("uploaded_files")

	setup.SetupWebsocket(r)

	r.Run()
}
