package endpoints

import "testing"

func TestDetectContentType(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/index.html", "text/html; charset=utf-8"},
		{"/page.htm", "text/html; charset=utf-8"},
		{"/style.css", "text/css; charset=utf-8"},
		{"/app.js", "application/javascript"},
		{"/app.mjs", "application/javascript"},
		{"/data.json", "application/json"},
		{"/site.webmanifest", "application/json"},
		{"/icon.svg", "image/svg+xml"},
		{"/feed.xml", "application/xml"},
		{"/module.wasm", "application/wasm"},
		{"/photo.png", "image/png"},
		{"/unknown.xyz", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := detectContentType(tt.path); got != tt.want {
				t.Fatalf("detectContentType(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestIsCompressible(t *testing.T) {
	compressible := []string{
		"text/html; charset=utf-8",
		"text/plain",
		"application/javascript",
		"application/json",
		"application/xml",
		"image/svg+xml",
		"application/wasm",
	}
	for _, ct := range compressible {
		if !isCompressible(ct) {
			t.Fatalf("isCompressible(%q) = false, want true", ct)
		}
	}

	notCompressible := []string{
		"image/png",
		"image/jpeg",
		"application/octet-stream",
		"video/mp4",
	}
	for _, ct := range notCompressible {
		if isCompressible(ct) {
			t.Fatalf("isCompressible(%q) = true, want false", ct)
		}
	}
}

func TestParseAcceptEncoding(t *testing.T) {
	prefs := parseAcceptEncoding("gzip, deflate, br;q=0.5")
	if !prefs["gzip"].present || prefs["gzip"].quality != 1.0 {
		t.Fatalf("gzip preference wrong: %+v", prefs["gzip"])
	}
	if !prefs["br"].present || prefs["br"].quality != 0.5 {
		t.Fatalf("br preference wrong: %+v", prefs["br"])
	}
	if prefs["deflate"].present == false {
		t.Fatalf("deflate should be present with default quality")
	}
}

func TestParseAcceptEncodingEmpty(t *testing.T) {
	if got := parseAcceptEncoding(""); len(got) != 0 {
		t.Fatalf("expected empty map for empty header, got %v", got)
	}
}

func TestNegotiateEncoding(t *testing.T) {
	cached := CachedFile{
		Raw:    []byte("raw"),
		Gzip:   []byte("gzip"),
		Brotli: []byte("brotli"),
	}

	tests := []struct {
		name         string
		acceptHeader string
		wantEncoding string
		wantBody     []byte
	}{
		{"prefers brotli", "gzip, br", "br", cached.Brotli},
		{"prefers gzip when br lower quality", "gzip;q=1.0, br;q=0.5", "gzip", cached.Gzip},
		{"gzip only", "gzip", "gzip", cached.Gzip},
		{"br only", "br", "br", cached.Brotli},
		{"no encoding accepted", "identity", "", cached.Raw},
		{"empty header", "", "", cached.Raw},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoding, body := negotiateEncoding(tt.acceptHeader, cached)
			if encoding != tt.wantEncoding {
				t.Fatalf("encoding = %q, want %q", encoding, tt.wantEncoding)
			}
			if string(body) != string(tt.wantBody) {
				t.Fatalf("body = %q, want %q", body, tt.wantBody)
			}
		})
	}
}

func TestNegotiateEncodingUncompressedFile(t *testing.T) {
	cached := CachedFile{Raw: []byte("raw")}
	encoding, body := negotiateEncoding("gzip, br", cached)
	if encoding != "" || string(body) != "raw" {
		t.Fatalf("uncompressed file should return raw body, got encoding=%q body=%q", encoding, body)
	}
}
