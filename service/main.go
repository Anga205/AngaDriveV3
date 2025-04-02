package main

import (
	"service/sysinfo"
)

func main() {
	// r := gin.Default()
	// setup.SetupFrontend(r)
	// setup.SetupDB(global.UploadedFilesDir)

	sysinfo.GetSysInfo()

	// r.Run()
}
