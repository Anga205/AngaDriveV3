package vars

import (
	"os"
)

var FrontendURL string
var BackendURL string

func init() {
	FrontendURL = os.Getenv("FRONTEND_URL")
	BackendURL = os.Getenv("BACKEND_URL")

	if FrontendURL == "" {
		FrontendURL = "localhost:8080"
	}

	if BackendURL == "" {
		BackendURL = "localhost:8080"
	}
}
