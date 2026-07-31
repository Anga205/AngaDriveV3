package endpoints

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func benchmarkChunkSequence(sizeMB int, chunkSize int) ([]byte, [][]byte) {
	payload := bytes.Repeat([]byte("angadrive-streaming-upload-benchmark-"), 1024*1024/len("angadrive-streaming-upload-benchmark-"))
	payload = bytes.Repeat(payload, sizeMB)
	compressedChunks := make([][]byte, 0)
	for start := 0; start < len(payload); start += chunkSize {
		end := start + chunkSize
		if end > len(payload) {
			end = len(payload)
		}
		chunk := payload[start:end]
		var buf bytes.Buffer
		zw := gzip.NewWriter(&buf)
		_, err := zw.Write(chunk)
		if err != nil {
			panic(err)
		}
		if err := zw.Close(); err != nil {
			panic(err)
		}
		compressedChunks = append(compressedChunks, buf.Bytes())
	}
	return payload, compressedChunks
}

func BenchmarkLegacyChunkAssembly(b *testing.B) {
	payload, compressedChunks := benchmarkChunkSequence(8, 7*1024*1024)
	b.ReportAllocs()

	for b.Loop() {
		baseDir := b.TempDir()
		for j, chunk := range compressedChunks {
			path := filepath.Join(baseDir, fmt.Sprintf("%d.part", j))
			if err := os.WriteFile(path, chunk, 0o644); err != nil {
				b.Fatal(err)
			}
		}
		combined := bytes.NewBuffer(nil)
		for j := 0; j < len(compressedChunks); j++ {
			path := filepath.Join(baseDir, fmt.Sprintf("%d.part", j))
			data, err := os.ReadFile(path)
			if err != nil {
				b.Fatal(err)
			}
			combined.Write(data)
		}
		gr, err := gzip.NewReader(bytes.NewReader(combined.Bytes()))
		if err != nil {
			b.Fatal(err)
		}
		if _, err := io.ReadAll(gr); err != nil {
			b.Fatal(err)
		}
		_ = payload
	}
}

func BenchmarkStreamingGzipAssembly(b *testing.B) {
	payload, compressedChunks := benchmarkChunkSequence(8, 7*1024*1024)
	b.ReportAllocs()

	for b.Loop() {
		baseDir := b.TempDir()
		for j, chunk := range compressedChunks {
			path := filepath.Join(baseDir, fmt.Sprintf("%d.part", j))
			if err := os.WriteFile(path, chunk, 0o644); err != nil {
				b.Fatal(err)
			}
		}
		assembledPath, err := assembleGzipStream(baseDir, len(compressedChunks))
		if err != nil {
			b.Fatal(err)
		}
		defer os.Remove(assembledPath)
		f, err := os.Open(assembledPath)
		if err != nil {
			b.Fatal(err)
		}
		gr, err := gzip.NewReader(f)
		if err != nil {
			f.Close()
			b.Fatal(err)
		}
		if _, err := io.ReadAll(gr); err != nil {
			gr.Close()
			f.Close()
			b.Fatal(err)
		}
		gr.Close()
		f.Close()
		_ = payload
	}
}
