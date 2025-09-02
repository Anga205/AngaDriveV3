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
	// FOR DEVELOPMENT ONLY
	// if gin.Mode() != gin.ReleaseMode {
	// 	r.Use(func(c *gin.Context) {
	// 		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	// 		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
	// 		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
	// 		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

	// 		if c.Request.Method == "OPTIONS" {
	// 			c.AbortWithStatus(204)
	// 			return
	// 		}

	// 		c.Next()
	// 	})
	// }

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
