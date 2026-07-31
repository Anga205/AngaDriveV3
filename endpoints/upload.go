package endpoints

import (
	"compress/gzip"
	"crypto/md5"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"angadrive/accounts"
	"angadrive/database"
	"angadrive/socketHandler"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const timeout = 5 * time.Minute

var (
	uploadTimers = make(map[string]*time.Timer)
	timerLock    sync.Mutex
	chunkDir     string
	UPLOAD_DIR   string
)

const (
	uploadFrameChunk = byte(1)
)

type uploadChunkFrame struct {
	UploadID   string
	FileID     string
	Encoding   string
	ChunkIndex int
	Checksum   string
	Payload    []byte
}

func encodeUploadChunkFrame(uploadID, fileID, encoding string, chunkIndex int, payload []byte) []byte {
	checksum := sha256.Sum256(payload)
	checksumHex := hex.EncodeToString(checksum[:])

	headerLen := 1 + 4 + 8 + 4 + 4 + 4 + 64
	uploadIDBytes := []byte(uploadID)
	fileIDBytes := []byte(fileID)
	encodingBytes := []byte(encoding)
	frame := make([]byte, headerLen+len(uploadIDBytes)+len(fileIDBytes)+len(encodingBytes)+len(payload))
	frame[0] = uploadFrameChunk
	binary.LittleEndian.PutUint32(frame[1:5], uint32(chunkIndex))
	binary.LittleEndian.PutUint64(frame[5:13], uint64(len(payload)))
	binary.LittleEndian.PutUint32(frame[13:17], uint32(len(uploadIDBytes)))
	binary.LittleEndian.PutUint32(frame[17:21], uint32(len(fileIDBytes)))
	binary.LittleEndian.PutUint32(frame[21:25], uint32(len(encodingBytes)))
	copy(frame[25:89], []byte(checksumHex))
	offset := 89
	copy(frame[offset:offset+len(uploadIDBytes)], uploadIDBytes)
	offset += len(uploadIDBytes)
	copy(frame[offset:offset+len(fileIDBytes)], fileIDBytes)
	offset += len(fileIDBytes)
	copy(frame[offset:offset+len(encodingBytes)], encodingBytes)
	offset += len(encodingBytes)
	copy(frame[offset:offset+len(payload)], payload)
	return frame
}

func decodeUploadChunkFrame(frame []byte) (uploadChunkFrame, error) {
	if len(frame) < 89 {
		return uploadChunkFrame{}, fmt.Errorf("frame too short")
	}
	if frame[0] != uploadFrameChunk {
		return uploadChunkFrame{}, fmt.Errorf("unsupported frame type")
	}
	chunkIndex := int(binary.LittleEndian.Uint32(frame[1:5]))
	payloadLen := int(binary.LittleEndian.Uint64(frame[5:13]))
	uploadIDLen := int(binary.LittleEndian.Uint32(frame[13:17]))
	fileIDLen := int(binary.LittleEndian.Uint32(frame[17:21]))
	encodingLen := int(binary.LittleEndian.Uint32(frame[21:25]))
	checksumHex := string(frame[25:89])
	offset := 89
	if offset+uploadIDLen+fileIDLen+encodingLen > len(frame) {
		return uploadChunkFrame{}, fmt.Errorf("frame metadata truncated")
	}
	uploadID := string(frame[offset : offset+uploadIDLen])
	offset += uploadIDLen
	fileID := string(frame[offset : offset+fileIDLen])
	offset += fileIDLen
	encoding := string(frame[offset : offset+encodingLen])
	offset += encodingLen
	payloadEnd := offset + payloadLen
	if payloadEnd > len(frame) {
		return uploadChunkFrame{}, fmt.Errorf("payload truncated")
	}
	payload := append([]byte(nil), frame[offset:payloadEnd]...)
	checksumActual := sha256.Sum256(payload)
	if hex.EncodeToString(checksumActual[:]) != strings.TrimSpace(checksumHex) {
		return uploadChunkFrame{}, fmt.Errorf("chunk checksum mismatch")
	}
	return uploadChunkFrame{UploadID: uploadID, FileID: fileID, Encoding: encoding, ChunkIndex: chunkIndex, Checksum: checksumHex, Payload: payload}, nil
}

func handleChunkUpload(c *gin.Context) {
	uploadID := c.Param("uuid")
	chunkIndexStr := c.PostForm("chunkIndex")
	encoding := c.PostForm("encoding")
	chunkSizeStr := c.PostForm("chunkSize")
	if chunkIndexStr == "" {
		c.String(400, "Missing chunkIndex")
		return
	}

	file, _, err := c.Request.FormFile("chunk")
	if err != nil {
		c.String(400, "Missing chunk")
		return
	}
	defer file.Close()

	uploadPath := filepath.Join(chunkDir, uploadID)
	if err := os.MkdirAll(uploadPath, os.ModePerm); err != nil {
		c.String(500, "Failed to create upload directory")
		return
	}

	chunkPath := filepath.Join(uploadPath, fmt.Sprintf("%s.part", chunkIndexStr))
	if encoding == "gzip-stream-v1" {
		if chunkSizeStr != "" {
			if expected, err := strconv.Atoi(chunkSizeStr); err == nil {
				if expected <= 0 {
					c.String(400, "Invalid chunkSize")
					return
				}
				if _, err := file.Seek(0, io.SeekStart); err != nil {
					c.String(500, "Failed to reset chunk stream")
					return
				}
				chunkBytes, err := io.ReadAll(file)
				if err != nil {
					c.String(500, "Failed to read chunk bytes")
					return
				}
				if len(chunkBytes) != expected {
					c.String(400, fmt.Sprintf("Chunk size mismatch: expected %d bytes got %d", expected, len(chunkBytes)))
					return
				}
				if err := os.WriteFile(chunkPath, chunkBytes, 0o644); err != nil {
					c.String(500, "Failed to write chunk file")
					return
				}
			} else {
				c.String(400, "Invalid chunkSize")
				return
			}
		} else {
			out, err := os.Create(chunkPath)
			if err != nil {
				c.String(500, "Failed to create chunk file")
				return
			}
			defer out.Close()
			if _, err := io.Copy(out, file); err != nil {
				c.String(500, "Failed to store chunk file")
				return
			}
		}
		resetUploadTimer(uploadID)
		c.String(200, "Chunk received")
		return
	}

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		c.String(400, "Could not decompress chunk. Make sure it's gzipped.")
		return
	}
	defer gzReader.Close()

	out, err := os.Create(chunkPath)
	if err != nil {
		c.String(500, "Failed to create chunk file")
		return
	}
	defer out.Close()
	if _, err := io.Copy(out, gzReader); err != nil {
		c.String(500, "Failed to write chunk file")
		return
	}

	resetUploadTimer(uploadID)
	c.String(200, "Chunk received")
}

func handleUploadWebSocket(c *gin.Context) {
	connUploadID := c.Param("uuid")
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upgrade upload websocket"})
		return
	}
	defer conn.Close()

	if connUploadID != "pool" && connUploadID != "" {
		uploadPath := filepath.Join(chunkDir, connUploadID)
		_ = os.MkdirAll(uploadPath, os.ModePerm)
	}

	for {
		messageType, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}
		switch messageType {
		case websocket.TextMessage:
			var control map[string]any
			if err := json.Unmarshal(msg, &control); err != nil {
				_ = conn.WriteJSON(map[string]any{"type": "UPLOAD_ERROR", "message": "Invalid control frame"})
				continue
			}
			switch control["type"] {
			case "INIT", "FILE_START", "PAUSE", "RESUME", "PING":
				if control["type"] == "PING" {
					_ = conn.WriteJSON(map[string]any{"type": "PONG"})
				}
			case "FINALIZE":
				finalizeControl := control
				targetUploadID, _ := finalizeControl["uploadId"].(string)
				if targetUploadID == "" {
					targetUploadID = connUploadID
				}
				if targetUploadID == "" || targetUploadID == "pool" {
					_ = conn.WriteJSON(map[string]any{"type": "UPLOAD_ERROR", "message": "Missing uploadId for FINALIZE"})
					continue
				}

				targetUploadPath := filepath.Join(chunkDir, targetUploadID)

				var authToken, authEmail, authPassword string
				if val, ok := finalizeControl["auth"]; ok {
					if authObj, ok := val.(map[string]any); ok {
						authToken, _ = authObj["token"].(string)
						authEmail, _ = authObj["email"].(string)
						authPassword, _ = authObj["password"].(string)
					}
				}
				originalFileName, _ := finalizeControl["fileName"].(string)
				fileID, _ := finalizeControl["fileId"].(string)
				collectionID, _ := finalizeControl["collectionId"].(string)
				totalChunks, _ := strconv.Atoi(fmt.Sprintf("%v", finalizeControl["totalChunks"]))
				fileSize, _ := strconv.ParseInt(fmt.Sprintf("%v", finalizeControl["fileSize"]), 10, 64)
				encodingName, _ := finalizeControl["encoding"].(string)
				if originalFileName == "" {
					_ = conn.WriteJSON(map[string]any{"type": "UPLOAD_ERROR", "uploadId": targetUploadID, "fileId": fileID, "message": "Missing originalFileName"})
					continue
				}
				if totalChunks <= 0 {
					_ = conn.WriteJSON(map[string]any{"type": "UPLOAD_ERROR", "uploadId": targetUploadID, "fileId": fileID, "message": "Missing totalChunks"})
					continue
				}
				if authToken == "" && (authEmail == "" || authPassword == "") {
					_ = conn.WriteJSON(map[string]any{"type": "UPLOAD_ERROR", "uploadId": targetUploadID, "fileId": fileID, "message": "Missing authentication details"})
					continue
				}

				_ = conn.WriteJSON(map[string]any{
					"type":        "DATA_RECEIVED",
					"uploadId":    targetUploadID,
					"fileId":      fileID,
					"totalChunks": totalChunks,
				})

				finalFilePath, fileData, err := finalizeUploadFromSession(targetUploadID, targetUploadPath, totalChunks, originalFileName, collectionID, encodingName, strconv.FormatInt(fileSize, 10), authToken, authEmail, authPassword)
				if err != nil {
					_ = conn.WriteJSON(map[string]any{"type": "UPLOAD_ERROR", "uploadId": targetUploadID, "fileId": fileID, "message": err.Error()})
					continue
				}
				_ = conn.WriteJSON(map[string]any{
					"type":     "FILE_COMPLETE",
					"uploadId": targetUploadID,
					"fileId":   fileID,
					"filePath": finalFilePath,
					"file":     fileData,
				})
			default:
				_ = conn.WriteJSON(map[string]any{"type": "UPLOAD_ERROR", "message": "Unknown control message"})
			}
		case websocket.BinaryMessage:
			frame, err := decodeUploadChunkFrame(msg)
			if err != nil {
				_ = conn.WriteJSON(map[string]any{"type": "UPLOAD_ERROR", "message": err.Error()})
				continue
			}

			targetUploadID := frame.UploadID
			if targetUploadID == "" {
				targetUploadID = connUploadID
			}
			if targetUploadID == "" || targetUploadID == "pool" {
				_ = conn.WriteJSON(map[string]any{"type": "UPLOAD_ERROR", "message": "Missing uploadId in frame"})
				continue
			}

			targetUploadPath := filepath.Join(chunkDir, targetUploadID)
			if err := os.MkdirAll(targetUploadPath, os.ModePerm); err != nil {
				_ = conn.WriteJSON(map[string]any{"type": "UPLOAD_ERROR", "uploadId": targetUploadID, "fileId": frame.FileID, "message": "Failed to create upload directory"})
				continue
			}

			if frame.Encoding == "" {
				frame.Encoding = "gzip-stream-v1"
			}
			chunkPath := filepath.Join(targetUploadPath, fmt.Sprintf("%d.part", frame.ChunkIndex))
			if err := os.WriteFile(chunkPath, frame.Payload, 0o644); err != nil {
				_ = conn.WriteJSON(map[string]any{"type": "UPLOAD_ERROR", "uploadId": targetUploadID, "fileId": frame.FileID, "message": "Failed to persist chunk"})
				continue
			}
			resetUploadTimer(targetUploadID)
			_ = conn.WriteJSON(map[string]any{
				"type":           "CHUNK_ACK",
				"uploadId":       targetUploadID,
				"fileId":         frame.FileID,
				"chunkIndex":     frame.ChunkIndex,
				"payloadBytes":   len(frame.Payload),
				"allChunksAcked": false,
			})
		}
	}
}

func finalizeUploadFromSession(uploadID, uploadPath string, totalChunks int, originalFileName, collectionID, encoding, expectedFileSizeStr, authToken, authEmail, authPassword string) (string, database.FileData, error) {
	if originalFileName == "" {
		return "", database.FileData{}, fmt.Errorf("missing originalFileName")
	}
	if totalChunks <= 0 {
		return "", database.FileData{}, fmt.Errorf("invalid totalChunks")
	}
	if authToken == "" && (authEmail == "" || authPassword == "") {
		return "", database.FileData{}, fmt.Errorf("missing authentication details")
	}

	accountToken := authToken
	if accountToken == "" {
		if accounts.Authenticate(authEmail, authPassword) {
			user, err := database.FindUserByEmail(authEmail)
			if err != nil {
				return "", database.FileData{}, fmt.Errorf("authentication successful but failed to retrieve user details")
			}
			accountToken = user.Token
		} else {
			return "", database.FileData{}, fmt.Errorf("invalid email or password")
		}
	}

	missingChunks := checkMissingChunks(uploadPath, totalChunks)
	if len(missingChunks) > 0 {
		return "", database.FileData{}, fmt.Errorf("some chunks are missing: %v", missingChunks)
	}

	finalDestDir := filepath.Join(UPLOAD_DIR, "i")
	if err := os.MkdirAll(finalDestDir, os.ModePerm); err != nil {
		return "", database.FileData{}, fmt.Errorf("failed to create destination directory")
	}

	tempFile, err := os.CreateTemp(finalDestDir, "upload-*.tmp")
	if err != nil {
		return "", database.FileData{}, fmt.Errorf("failed to create temporary file")
	}
	tempFilePath := tempFile.Name()
	defer func() {
		_ = tempFile.Close()
		_ = os.Remove(tempFilePath)
	}()

	hash := md5.New()
	multiOut := io.MultiWriter(tempFile, hash)

	if encoding == "gzip-stream-v1" {
		chunkReaders := make([]io.Reader, totalChunks)
		chunkFiles := make([]*os.File, totalChunks)
		for i := 0; i < totalChunks; i++ {
			chunkPath := filepath.Join(uploadPath, fmt.Sprintf("%d.part", i))
			f, err := os.Open(chunkPath)
			if err != nil {
				for k := 0; k < i; k++ {
					_ = chunkFiles[k].Close()
				}
				return "", database.FileData{}, fmt.Errorf("failed to open chunk %d: %w", i, err)
			}
			chunkFiles[i] = f
			chunkReaders[i] = f
		}
		defer func() {
			for _, f := range chunkFiles {
				if f != nil {
					_ = f.Close()
				}
			}
		}()

		multiReader := io.MultiReader(chunkReaders...)
		gzReader, err := gzip.NewReader(multiReader)
		if err != nil {
			return "", database.FileData{}, fmt.Errorf("could not initialize gzip reader: %w", err)
		}
		defer gzReader.Close()

		if _, err := io.Copy(multiOut, gzReader); err != nil {
			return "", database.FileData{}, fmt.Errorf("failed to decompress assembled gzip stream: %w", err)
		}
	} else {
		for i := 0; i < totalChunks; i++ {
			chunkPath := filepath.Join(uploadPath, fmt.Sprintf("%d.part", i))
			chunkFile, err := os.Open(chunkPath)
			if err != nil {
				return "", database.FileData{}, fmt.Errorf("failed to open chunk %d: %w", i, err)
			}
			_, err = io.Copy(multiOut, chunkFile)
			chunkFile.Close()
			if err != nil {
				return "", database.FileData{}, fmt.Errorf("failed to copy chunk %d: %w", i, err)
			}
		}
	}

	md5sum := hex.EncodeToString(hash.Sum(nil))
	fileInfo, err := tempFile.Stat()
	if err != nil {
		return "", database.FileData{}, fmt.Errorf("failed to get final file stats")
	}
	fileSize := fileInfo.Size()
	if expectedFileSizeStr != "" {
		if expectedSize, err := strconv.ParseInt(expectedFileSizeStr, 10, 64); err == nil && expectedSize > 0 && expectedSize != fileSize {
			return "", database.FileData{}, fmt.Errorf("file size mismatch: expected %d bytes got %d", expectedSize, fileSize)
		}
	}

	finalFilePath := filepath.Join(finalDestDir, md5sum+filepath.Ext(originalFileName))
	_ = tempFile.Close()
	if err := os.Rename(tempFilePath, finalFilePath); err != nil {
		return "", database.FileData{}, fmt.Errorf("failed to rename temporary file: %w", err)
	}

	uniqueFileName := database.GenerateUniqueFileName(originalFileName)
	fileData := database.FileData{
		OriginalFileName: originalFileName,
		FileDirectory:    uniqueFileName,
		AccountToken:     accountToken,
		FileSize:         fileSize,
		Timestamp:        time.Now().Unix(),
		Md5sum:           md5sum + filepath.Ext(originalFileName),
	}

	if err := fileData.Insert(); err != nil {
		_ = os.Remove(finalFilePath)
		return "", database.FileData{}, fmt.Errorf("failed to insert file metadata: %w", err)
	}

	if collectionID != "" {
		addReq := socketHandler.AddFileToCollectionRequest{
			CollectionID:  collectionID,
			FileDirectory: fileData.FileDirectory,
			Auth:          socketHandler.AuthInfo{Token: accountToken},
		}
		if _, err := socketHandler.AddFileToCollection(addReq); err != nil {
			fmt.Printf("Warning: Failed to add file %s to collection %s: %v\n", fileData.FileDirectory, collectionID, err)
		}
	}

	timerLock.Lock()
	if timer, ok := uploadTimers[uploadID]; ok {
		timer.Stop()
		delete(uploadTimers, uploadID)
	}
	timerLock.Unlock()

	if err := os.RemoveAll(uploadPath); err != nil {
		fmt.Printf("Warning: Failed to remove chunk directory %s: %v\n", uploadPath, err)
	}

	var FileUpdate socketHandler.FileUpdate
	FileUpdate.File = fileData
	FileUpdate.Toggle = true
	go socketHandler.UserFilesPulse(FileUpdate)
	go socketHandler.UpdateUserCount()

	return finalFilePath, fileData, nil
}

func finalizeUpload(c *gin.Context) {
	uploadID := c.Param("uuid")
	totalChunksStr := c.PostForm("totalChunks")
	originalFileName := c.PostForm("originalFileName")
	collectionID := c.PostForm("collectionId")
	encoding := c.PostForm("encoding")
	expectedFileSizeStr := c.PostForm("fileSize")
	expectedHash := c.PostForm("sha256")

	userToken := c.PostForm("token")
	email := c.PostForm("email")
	password := c.PostForm("password")

	if originalFileName == "" {
		c.String(400, "Missing originalFileName")
		return
	}

	var accountToken string
	if email != "" && password != "" {
		if accounts.Authenticate(email, password) {
			user, err := database.FindUserByEmail(email)
			if err != nil {
				c.String(401, "Authentication successful but failed to retrieve user details")
				return
			}
			accountToken = user.Token
		} else {
			c.String(401, "Invalid email or password")
			return
		}
	} else if userToken != "" {
		accountToken = userToken
	} else {
		c.String(400, "Missing authentication details (token or email/password)")
		return
	}

	totalChunks, err := strconv.Atoi(totalChunksStr)
	if err != nil {
		c.String(400, "Invalid totalChunks")
		return
	}
	if totalChunks <= 0 {
		c.String(400, "Invalid totalChunks")
		return
	}

	uploadPath := filepath.Join(chunkDir, uploadID)
	finalFilePath, fileData, err := finalizeUploadFromSession(uploadID, uploadPath, totalChunks, originalFileName, collectionID, encoding, expectedFileSizeStr, accountToken, "", "")
	if err != nil {
		c.String(400, err.Error())
		return
	}
	if expectedHash != "" {
		tempFile, err := os.Open(finalFilePath)
		if err != nil {
			c.String(500, "Failed to open final file")
			return
		}
		defer tempFile.Close()
		hashWriter := sha256.New()
		if _, err := io.Copy(hashWriter, tempFile); err != nil {
			c.String(500, "Failed to compute SHA-256 hash")
			return
		}
		if expectedHash != hex.EncodeToString(hashWriter.Sum(nil)) {
			c.String(400, "File integrity mismatch")
			return
		}
	}

	c.JSON(200, gin.H{
		"message":       "Upload successful and file assembled",
		"fileName":      fileData.FileDirectory,
		"fileDirectory": fileData.FileDirectory,
		"accessPath":    fmt.Sprintf("/i/%s", fileData.FileDirectory),
	})
}

func assembleGzipStream(uploadPath string, totalChunks int) (string, error) {
	assembledFile, err := os.CreateTemp(uploadPath, "assembled-*.gz")
	if err != nil {
		return "", err
	}
	defer assembledFile.Close()

	for i := 0; i < totalChunks; i++ {
		chunkPath := filepath.Join(uploadPath, fmt.Sprintf("%d.part", i))
		chunk, err := os.Open(chunkPath)
		if err != nil {
			os.Remove(assembledFile.Name())
			return "", err
		}
		if _, err := io.Copy(assembledFile, chunk); err != nil {
			chunk.Close()
			os.Remove(assembledFile.Name())
			return "", err
		}
		chunk.Close()
	}

	if _, err := assembledFile.Seek(0, io.SeekStart); err != nil {
		os.Remove(assembledFile.Name())
		return "", err
	}
	return assembledFile.Name(), nil
}

func computeUploadProgress(acknowledgedBytes, totalBytes int64, fileComplete bool) int {
	if totalBytes <= 0 {
		return 0
	}
	if acknowledgedBytes <= 0 {
		return 0
	}
	progress := int((float64(acknowledgedBytes) / float64(totalBytes)) * 100)
	if progress >= 100 || fileComplete {
		return 100
	}
	return progress
}

func checkMissingChunks(uploadPath string, totalChunks int) []int {
	var missingChunks []int
	for i := 0; i < totalChunks; i++ {
		chunkPath := filepath.Join(uploadPath, fmt.Sprintf("%d.part", i))
		if _, err := os.Stat(chunkPath); os.IsNotExist(err) {
			missingChunks = append(missingChunks, i)
		}
	}
	return missingChunks
}

func resetUploadTimer(uploadID string) {
	timerLock.Lock()
	defer timerLock.Unlock()

	if timer, ok := uploadTimers[uploadID]; ok {
		timer.Reset(timeout)
	} else {
		uploadTimers[uploadID] = time.AfterFunc(timeout, func() {
			os.RemoveAll(filepath.Join(chunkDir, uploadID))
			timerLock.Lock()
			delete(uploadTimers, uploadID)
			timerLock.Unlock()
			fmt.Printf("Upload %s expired and deleted\n", uploadID)
		})
	}
}

func setupUploaderRoutes(r *gin.Engine, UPLOAD_DIR_BASE string) {
	chunkDir = UPLOAD_DIR_BASE + "/tmp_chunks"
	UPLOAD_DIR = UPLOAD_DIR_BASE
	os.RemoveAll(chunkDir)
	os.MkdirAll(chunkDir, os.ModePerm)
	r.GET("/upload/ws/:uuid", handleUploadWebSocket)
	r.POST("/upload/:uuid", handleChunkUpload)
	r.POST("/upload/success/:uuid", finalizeUpload)
}
