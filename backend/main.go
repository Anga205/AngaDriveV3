package main

import (
	"service/database"
	"service/endpoints"
	"service/info"
	"service/socketHandler"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

func main() {

	UPLOAD_DIR := "uploaded_files"

	r := gin.Default()
	r.Use(gzip.Gzip(gzip.BestCompression))

	err := endpoints.SetupFrontend(r)
	if err != nil {
		panic(err)
	}
	database.InitializeDatabase(UPLOAD_DIR)
	info.GetSpaceUsedGraph()
	socketHandler.SetupWebsocket(r, UPLOAD_DIR)
	endpoints.InitEndpoints(r, UPLOAD_DIR)

	r.Run()
}
