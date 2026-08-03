package endpoints

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// newUploadTestRouter builds a gin router with only the chunk-upload route
// wired up, using a temporary chunk directory.
func newUploadTestRouter(t *testing.T) (*gin.Engine, string) {
	t.Helper()
	dir := t.TempDir()
	chunkDir = filepath.Join(dir, "tmp_chunks")
	UPLOAD_DIR = dir
	if err := os.MkdirAll(chunkDir, os.ModePerm); err != nil {
		t.Fatalf("failed to create chunkDir: %v", err)
	}

	r := gin.New()
	r.POST("/upload/:uuid", handleChunkUpload)
	return r, dir
}

// gzipBytes compresses b with gzip.
func gzipBytes(t *testing.T, b []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write(b); err != nil {
		t.Fatalf("gzip write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}
	return buf.Bytes()
}

// postChunk sends a multipart chunk upload request for the given encoding and
// payload bytes, returning the HTTP response.
func postChunk(t *testing.T, r *gin.Engine, uuid, chunkIndex, encoding string, payload []byte, filename string) *httptest.ResponseRecorder {
	t.Helper()

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)

	// encoding form field
	if err := mw.WriteField("encoding", encoding); err != nil {
		t.Fatalf("write encoding field: %v", err)
	}
	if err := mw.WriteField("chunkIndex", chunkIndex); err != nil {
		t.Fatalf("write chunkIndex field: %v", err)
	}

	// chunk file part
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="chunk"; filename="`+filename+`"`)
	h.Set("Content-Type", "application/octet-stream")
	fw, err := mw.CreatePart(h)
	if err != nil {
		t.Fatalf("create part: %v", err)
	}
	if _, err := fw.Write(payload); err != nil {
		t.Fatalf("write chunk payload: %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/upload/"+uuid, &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestResolveEncoding(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		want   string
		wantOK bool
	}{
		{"empty falls back to gzip", "", EncodingGzip, true},
		{"explicit gzip", "gzip-stream-v1", EncodingGzip, true},
		{"explicit raw", "raw", EncodingRaw, true},
		{"unknown encoding rejected", "lzma", "", false},
		{"bogus encoding rejected", "base64", "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := resolveEncoding(tc.input)
			if ok != tc.wantOK {
				t.Fatalf("resolveEncoding(%q) ok = %v, want %v", tc.input, ok, tc.wantOK)
			}
			if got != tc.want {
				t.Fatalf("resolveEncoding(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestUploadSessionEncodingAssociation(t *testing.T) {
	const id = "test-session"

	if _, found := getUploadEncoding(id); found {
		t.Fatal("expected no encoding before set")
	}
	setUploadEncoding(id, EncodingRaw)
	got, found := getUploadEncoding(id)
	if !found || got != EncodingRaw {
		t.Fatalf("expected raw encoding, got %q (found=%v)", got, found)
	}
	deleteUploadEncoding(id)
	if _, found := getUploadEncoding(id); found {
		t.Fatal("expected encoding to be deleted")
	}
}

func TestHandleChunkUploadGzip(t *testing.T) {
	r, dir := newUploadTestRouter(t)
	original := []byte("hello gzip chunk upload world")

	// Upload 2 gzip-compressed chunks.
	payload := gzipBytes(t, original)
	rec := postChunk(t, r, "gzip-session", "0", "gzip-stream-v1", payload, "file.txt.gz")
	if rec.Code != http.StatusOK {
		t.Fatalf("gzip chunk 0 status = %d, body = %s", rec.Code, rec.Body.String())
	}
	rec = postChunk(t, r, "gzip-session", "1", "gzip-stream-v1", gzipBytes(t, []byte(" part two")), "file.txt.gz")
	if rec.Code != http.StatusOK {
		t.Fatalf("gzip chunk 1 status = %d, body = %s", rec.Code, rec.Body.String())
	}

	// The stored .part files must contain the DECOMPRESSED bytes.
	part0, err := os.ReadFile(filepath.Join(chunkDir, "gzip-session", "0.part"))
	if err != nil {
		t.Fatalf("read part 0: %v", err)
	}
	if !bytes.Equal(part0, original) {
		t.Fatalf("gzip part 0 = %q, want %q", part0, original)
	}
	_ = dir
}

func TestHandleChunkUploadRaw(t *testing.T) {
	r, _ := newUploadTestRouter(t)

	// Arbitrary binary data, including bytes that are NOT valid gzip magic.
	original := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0x00, 0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A}

	rec := postChunk(t, r, "raw-session", "0", "raw", original, "data.bin")
	if rec.Code != http.StatusOK {
		t.Fatalf("raw chunk 0 status = %d, body = %s", rec.Code, rec.Body.String())
	}

	part0, err := os.ReadFile(filepath.Join(chunkDir, "raw-session", "0.part"))
	if err != nil {
		t.Fatalf("read part 0: %v", err)
	}
	if !bytes.Equal(part0, original) {
		t.Fatalf("raw part 0 = %v, want %v", part0, original)
	}
}

func TestHandleChunkUploadEncodingMismatch(t *testing.T) {
	r, _ := newUploadTestRouter(t)

	// Establish the session as raw.
	if rec := postChunk(t, r, "mismatch-session", "0", "raw", []byte("raw bytes"), "a.bin"); rec.Code != http.StatusOK {
		t.Fatalf("initial raw chunk status = %d", rec.Code)
	}

	// Now attempt to upload a gzip chunk into the same session — must be rejected.
	rec := postChunk(t, r, "mismatch-session", "1", "gzip-stream-v1", gzipBytes(t, []byte("gzip bytes")), "a.bin.gz")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("mismatched gzip chunk status = %d, want 400; body = %s", rec.Code, rec.Body.String())
	}
}

func TestAssembleChunkFilesOrder(t *testing.T) {
	dir := t.TempDir()

	// Write parts out of order to prove assembly is index-ordered.
	writePart := func(name string, data string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	writePart("2.part", "three-")
	writePart("0.part", "one-")
	writePart("1.part", "two-")

	var buf bytes.Buffer
	if err := assembleChunkFiles(dir, 3, &buf); err != nil {
		t.Fatalf("assembleChunkFiles: %v", err)
	}

	want := "one-two-three-"
	if buf.String() != want {
		t.Fatalf("assembled = %q, want %q", buf.String(), want)
	}
}

func TestAssembleChunkFilesBinary(t *testing.T) {
	dir := t.TempDir()
	chunks := [][]byte{
		{0x00, 0x01, 0x02, 0x03},
		{0xFF, 0xFE, 0xFD, 0xFC},
		{0x89, 0x50, 0x4E, 0x47}, // PNG-like magic, must survive untouched
	}
	for i, c := range chunks {
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("%d.part", i)), c, 0o644); err != nil {
			t.Fatalf("write part %d: %v", i, err)
		}
	}

	var buf bytes.Buffer
	if err := assembleChunkFiles(dir, len(chunks), &buf); err != nil {
		t.Fatalf("assembleChunkFiles: %v", err)
	}

	var want []byte
	for _, c := range chunks {
		want = append(want, c...)
	}
	if !bytes.Equal(buf.Bytes(), want) {
		t.Fatalf("assembled = %v, want %v", buf.Bytes(), want)
	}
}

func TestChunkChecksumPreservedForBothModes(t *testing.T) {
	// The server does not compute chunk checksums itself; it validates chunk
	// integrity by requiring all chunks to be present (checkMissingChunks) and
	// by recomputing the whole-file MD5 at finalization. This test verifies that
	// the per-mode storage path preserves bytes such that a subsequent MD5 of
	// the assembled file matches the original input for both encodings.

	r, dir := newUploadTestRouter(t)
	original := []byte("integrity check payload: 1234567890 !@#$%^&*()")

	// --- raw mode ---
	rawRouter := r
	_ = rawRouter
	if rec := postChunk(t, rawRouter, "raw-int", "0", "raw", original, "f.bin"); rec.Code != http.StatusOK {
		t.Fatalf("raw chunk status = %d", rec.Code)
	}
	rawPart, err := os.ReadFile(filepath.Join(chunkDir, "raw-int", "0.part"))
	if err != nil {
		t.Fatalf("read raw part: %v", err)
	}
	if !bytes.Equal(rawPart, original) {
		t.Fatalf("raw stored bytes mismatch")
	}

	// --- gzip mode ---
	if rec := postChunk(t, rawRouter, "gzip-int", "0", "gzip-stream-v1", gzipBytes(t, original), "f.txt.gz"); rec.Code != http.StatusOK {
		t.Fatalf("gzip chunk status = %d", rec.Code)
	}
	gzipPart, err := os.ReadFile(filepath.Join(chunkDir, "gzip-int", "0.part"))
	if err != nil {
		t.Fatalf("read gzip part: %v", err)
	}
	if !bytes.Equal(gzipPart, original) {
		t.Fatalf("gzip stored bytes mismatch (decompression failed or data altered)")
	}
	_ = dir
}
