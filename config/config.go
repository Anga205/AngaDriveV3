// Package config holds process-wide configuration resolved from the
// environment. It is the single place that reads environment variables so that
// the rest of the codebase depends on explicit values rather than on import
// side effects.
package config

import (
	"os"
	"time"
)

// Defaults used when the corresponding environment variable is unset.
const (
	defaultWebURL    = "localhost:8080"
	defaultAssetsURL = "localhost:8080"
	defaultUploadDir = "uploaded_files"
	defaultTimezone  = "Asia/Kolkata"
)

// Values resolved by Load. They are populated once during startup.
var (
	// WebURL is the host (host:port) that serves the application/websocket.
	WebURL string
	// AssetsURL is the host (host:port) that serves static assets and previews.
	AssetsURL string
	// UploadDir is the root directory for all persisted files.
	UploadDir string
	// Timezone is the IANA timezone used for activity/space-usage bucketing.
	Timezone string
)

func init() {
	Load()
}

// Load (re)reads configuration from the environment. It is safe to call
// multiple times; tests may call it after setting environment variables.
func Load() {
	WebURL = envOr("WEB_URL", defaultWebURL)
	AssetsURL = envOr("ASSETS_URL", defaultAssetsURL)
	UploadDir = envOr("UPLOAD_DIR", defaultUploadDir)
	Timezone = envOr("TIMEZONE", defaultTimezone)
}

// Location returns the configured timezone location, falling back to UTC if it
// cannot be loaded.
func Location() *time.Location {
	loc, err := time.LoadLocation(Timezone)
	if err != nil {
		return time.UTC
	}
	return loc
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
