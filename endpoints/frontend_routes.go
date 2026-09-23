package endpoints

import (
	"angadrive/globals"
	"angadrive/requestHandler"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

// collectionHandler serves the SPA for the /collection and /collection/* paths.
// If a legacy 'id' query parameter is present it redirects with a permanent 301
// to the canonical path-based collection URL.
func collectionHandler(indexFile CachedFile) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Host != globals.WebURL {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		legacyID := c.Query("id")
		if legacyID != "" {
			// Split the space-separated navigation path into individual IDs.
			// c.Query returns the decoded value, so "+" and %20 already became
			// spaces and we use strings.Fields (which also drops empties).
			ids := strings.Fields(legacyID)
			if len(ids) > 0 {
				escaped := make([]string, 0, len(ids))
				for _, id := range ids {
					escaped = append(escaped, url.PathEscape(id))
				}
				c.Redirect(http.StatusMovedPermanently, "/collection/"+strings.Join(escaped, "/"))
				return
			}
		}

		// No legacy id (e.g. /collection or /collection/) -> serve the SPA.
		go requestHandler.SiteActivityPulse()
		serveCachedFile(c, indexFile)
	}
}

func setupRoutes(r *gin.Engine, cache map[string]CachedFile) {
	indexFile, ok := cache["/index.html"]
	if !ok {
		fmt.Println("Error caching files: missing /index.html")
		return
	}

	// Collection routes: /collection and /collection/*path. The catch-all also
	// covers /collection/ (empty trailing segment via *path == "/"). Both use
	// the same handler so a legacy ?id= query on /collection or /collection/
	// produces an HTTP 301 redirect to the canonical path-based URL.
	r.GET("/collection", collectionHandler(indexFile))
	r.GET("/collection/*path", collectionHandler(indexFile))

	routes := []string{"/", "/my_drive", "/my_collections", "/account"}
	for _, route := range routes {
		route := route
		r.GET(route, func(c *gin.Context) {
			if c.Request.Host == globals.WebURL {
				go requestHandler.SiteActivityPulse()
				serveCachedFile(c, indexFile)
			} else if route == "/" && c.Request.Host == globals.AssetsURL {
				scheme := "http"
				if c.Request.TLS != nil {
					scheme = "https"
				} else if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
					scheme = proto
				}
				c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("%s://%s/", scheme, globals.WebURL))
			} else {
				c.AbortWithStatus(http.StatusNotFound)
			}
		})
	}

	for relPath, cachedFile := range cache { // register all files in the dist directory
		relPath := relPath
		cachedFile := cachedFile
		if relPath != "/index.html" { // since index.html is handled separately, we dont want to register it again
			r.GET(relPath, func(c *gin.Context) {
				if c.Request.Host == globals.WebURL {
					serveCachedFile(c, cachedFile)
				} else {
					c.AbortWithStatus(http.StatusNotFound)
				}
			})
		}
	}
}
