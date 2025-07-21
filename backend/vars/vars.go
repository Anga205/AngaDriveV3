package vars

import (
	"os"
)

var WebURL string
var AssetsURL string

func init() {
	WebURL = os.Getenv("WEB_URL")
	AssetsURL = os.Getenv("ASSETS_URL")

	if WebURL == "" {
		WebURL = "localhost:8080"
	}

	if AssetsURL == "" {
		AssetsURL = "localhost:8080"
	}
}
