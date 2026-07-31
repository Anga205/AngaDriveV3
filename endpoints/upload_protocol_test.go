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

func TestAssembleGzipStreamUpload(t *testing.T) {
	payload := bytes.Repeat([]byte("AngaDrive upload pipeline test data "), 128*1024)
	var compressed bytes.Buffer
	zw := gzip.NewWriter(&compressed)
	if _, err := zw.Write(payload); err != nil {
		t.Fatalf("gzip write failed: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("gzip close failed: %v", err)
	}

	chunkData := compressed.Bytes()
	chunkCount := 4
	chunkSize := (len(chunkData) + chunkCount - 1) / chunkCount
	uploadPath := filepath.Join(t.TempDir(), "stream-upload")
	if err := os.MkdirAll(uploadPath, 0o755); err != nil {
		t.Fatalf("mkdir upload path: %v", err)
	}

	for i := 0; i < chunkCount; i++ {
		start := i * chunkSize
		if start >= len(chunkData) {
			break
		}
		end := start + chunkSize
		if end > len(chunkData) {
			end = len(chunkData)
		}
		chunkPath := filepath.Join(uploadPath, fmt.Sprintf("%d.part", i))
		if err := os.WriteFile(chunkPath, chunkData[start:end], 0o644); err != nil {
			t.Fatalf("write chunk %d: %v", i, err)
		}
	}

	assembledPath, err := assembleGzipStream(uploadPath, chunkCount)
	if err != nil {
		t.Fatalf("assembleGzipStream failed: %v", err)
	}
	defer os.Remove(assembledPath)

	compressedFile, err := os.Open(assembledPath)
	if err != nil {
		t.Fatalf("open assembled gzip: %v", err)
	}
	defer compressedFile.Close()

	gr, err := gzip.NewReader(compressedFile)
	if err != nil {
		t.Fatalf("open gzip reader: %v", err)
	}
	defer gr.Close()

	decompressed, err := io.ReadAll(gr)
	if err != nil {
		t.Fatalf("read decompressed content: %v", err)
	}
	if !bytes.Equal(decompressed, payload) {
		t.Fatalf("payload mismatch: got %d bytes want %d", len(decompressed), len(payload))
	}
}

func TestCheckMissingChunksTracksResumeState(t *testing.T) {
	uploadPath := filepath.Join(t.TempDir(), "resume-upload")
	if err := os.MkdirAll(uploadPath, 0o755); err != nil {
		t.Fatalf("mkdir upload path: %v", err)
	}
	for _, idx := range []int{0, 2, 3} {
		if err := os.WriteFile(filepath.Join(uploadPath, fmt.Sprintf("%d.part", idx)), []byte("chunk"), 0o644); err != nil {
			t.Fatalf("write chunk %d: %v", idx, err)
		}
	}

	missing := checkMissingChunks(uploadPath, 5)
	if len(missing) != 2 || missing[0] != 1 || missing[1] != 4 {
		t.Fatalf("unexpected missing chunks: %#v", missing)
	}
}

func TestUploadProgressReaches100WhenDataReceivedOrComplete(t *testing.T) {
	progressBeforeComplete := computeUploadProgress(75, 100, false)
	if progressBeforeComplete >= 100 {
		t.Fatalf("progress should remain below 100 before server data received; got %d", progressBeforeComplete)
	}
	if progressBeforeComplete != 75 {
		t.Fatalf("expected 75 before completion, got %d", progressBeforeComplete)
	}
	progressDataReceived := computeUploadProgress(100, 100, false)
	if progressDataReceived != 100 {
		t.Fatalf("progress should reach 100 when all data is received; got %d", progressDataReceived)
	}
	progressAfterComplete := computeUploadProgress(100, 100, true)
	if progressAfterComplete != 100 {
		t.Fatalf("progress should reach 100 on FILE_COMPLETE; got %d", progressAfterComplete)
	}
}

func TestEncodeDecodeUploadChunkFrame(t *testing.T) {
	payload := []byte("multiplexed chunk test payload with binary data 1234567890")
	frameBytes := encodeUploadChunkFrame("upload-test-uuid-1", "file-test-uuid-1", "gzip-stream-v1", 3, payload)

	decoded, err := decodeUploadChunkFrame(frameBytes)
	if err != nil {
		t.Fatalf("decodeUploadChunkFrame failed: %v", err)
	}

	if decoded.UploadID != "upload-test-uuid-1" {
		t.Fatalf("expected uploadID 'upload-test-uuid-1', got '%s'", decoded.UploadID)
	}
	if decoded.FileID != "file-test-uuid-1" {
		t.Fatalf("expected fileID 'file-test-uuid-1', got '%s'", decoded.FileID)
	}
	if decoded.ChunkIndex != 3 {
		t.Fatalf("expected chunkIndex 3, got %d", decoded.ChunkIndex)
	}
	if !bytes.Equal(decoded.Payload, payload) {
		t.Fatalf("payload mismatch")
	}
}
