package main

import (
	"angadrive/database"
	"angadrive/endpoints"
	"angadrive/info"
	"angadrive/requestHandler"
	"fmt"

	"github.com/gin-gonic/gin"
)

func main() {

	r := gin.Default()
	// FOR DEVELOPMENT ONLY
	if gin.Mode() != gin.ReleaseMode {
		// TURN OFF CORS FOR DEVELOPMENT
		fmt.Println("=============== RUNNING IN DEV MODE, CORS DISABLED ===============")
		r.Use(func(c *gin.Context) {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

			if c.Request.Method == "OPTIONS" {
				c.AbortWithStatus(204)
				return
			}

			c.Next()
		})
	}

	err := endpoints.SetupFrontend(r)
	if err != nil {
		panic(err)
	}
	database.InitializeDatabase()
	info.GetSpaceUsedGraph()
	requestHandler.SetupWebsocket(r)
	endpoints.InitEndpoints(r)

	r.Run()
}
