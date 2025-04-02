package main

import (
	"service/setup"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	setup.SetupFrontend(r)
	setup.SetupDB("test.db")
	r.Run()
}
