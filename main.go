package main

import (
	"angadrive/config"
	"angadrive/database"
	"angadrive/endpoints"
	"angadrive/requestHandler"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	config.Load()

	r := gin.Default()
	// FOR DEVELOPMENT ONLY
	if gin.Mode() != gin.ReleaseMode {
		fmt.Println("=============== RUNNING IN DEV MODE, CORS DISABLED ===============")
		r.Use(devCORS())
	}

	if err := endpoints.SetupFrontend(r); err != nil {
		log.Fatalf("failed to set up frontend: %v", err)
	}
	if err := database.InitializeDatabase(); err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}
	requestHandler.SetupWebsocket(r)
	endpoints.InitEndpoints(r)

	if err := r.Run(); err != nil {
		log.Fatalf("server exited: %v", err)
	}
}

// devCORS allows all origins during development.
func devCORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
