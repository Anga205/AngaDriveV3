package main

import (
	"service/database"
	"service/setup"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	r.Use(gzip.Gzip(gzip.BestCompression))

	err := setup.SetupFrontend(r)
	if err != nil {
		panic(err)
	}
	database.InitializeDatabase("uploaded_files")

	setup.SetupWebsocket(r)

	r.Static("/i", "./uploaded_files/i")

	r.Run()
}
