package endpoints

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// minimalCachedIndex returns a cache containing just an /index.html marker so
// setupRoutes can register the SPA routes for testing.
func minimalCachedIndex() map[string]CachedFile {
	return map[string]CachedFile{
		"/index.html": {
			Raw:         []byte("<html>collection test</html>"),
			ContentType: "text/html; charset=utf-8",
		},
	}
}

func newCollectionTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	setupRoutes(r, minimalCachedIndex())
	return r
}

func doRequest(t *testing.T, router *gin.Engine, target string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	// vars.WebURL defaults to "localhost:8080" to match the host guard.
	req.Host = "localhost:8080"
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestCollectionQueryParamRedirects(t *testing.T) {
	router := newCollectionTestRouter()

	tests := []struct {
		name       string
		requestURL string
		wantStatus int
		wantLoc    string
	}{
		{"single id", "/collection/?id=A", http.StatusMovedPermanently, "/collection/A"},
		{"single id no trailing slash", "/collection?id=A", http.StatusMovedPermanently, "/collection/A"},
		{"multiple ids", "/collection/?id=A%20B%20C", http.StatusMovedPermanently, "/collection/A/B/C"},
		{"plus encoded ids", "/collection/?id=A+B+C", http.StatusMovedPermanently, "/collection/A/B/C"},
		{"arbitrary depth", "/collection/?id=A%20B%20C%20D%20E%20F", http.StatusMovedPermanently, "/collection/A/B/C/D/E/F"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := doRequest(t, router, tt.requestURL)
			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body=%s)", w.Code, tt.wantStatus, w.Body.String())
			}
			if got := w.Header().Get("Location"); got != tt.wantLoc {
				t.Fatalf("Location = %q, want %q", got, tt.wantLoc)
			}
		})
	}
}

func TestEmptyCollectionIdDoesNotRedirect(t *testing.T) {
	router := newCollectionTestRouter()

	w := doRequest(t, router, "/collection/?id=")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (empty id should serve the page, not redirect)", w.Code)
	}
}

func TestCollectionNoIdServesPage(t *testing.T) {
	router := newCollectionTestRouter()

	for _, url := range []string{"/collection", "/collection/", "/collection/A", "/collection/A/B/C"} {
		w := doRequest(t, router, url)
		if w.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d, want 200", url, w.Code)
		}
	}
}
