package main

import (
	"service/database"
	"service/setup"
	"service/uploader"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

func main() {

	UPLOAD_DIR := "uploaded_files"

	r := gin.Default()
	r.Use(gzip.Gzip(gzip.BestCompression))

	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowCredentials = true
	r.Use(cors.New(config))

	err := setup.SetupFrontend(r)
	if err != nil {
		panic(err)
	}
	database.InitializeDatabase(UPLOAD_DIR)

	setup.SetupWebsocket(r)
	uploader.SetupUploaderRoutes(r, UPLOAD_DIR)

	r.Static("/i", "./"+UPLOAD_DIR+"/i")

	r.Run()
}
