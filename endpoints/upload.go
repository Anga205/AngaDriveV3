package endpoints

import (
	"compress/gzip"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"angadrive/accounts"
	"angadrive/database"
	"angadrive/socketHandler"

	"github.com/gin-gonic/gin"
)

const timeout = 5 * time.Minute

// Supported upload encodings.
const (
	EncodingGzip = "gzip-stream-v1"
	EncodingRaw  = "raw"
)

var (
	uploadTimers    = make(map[string]*time.Timer)
	uploadEncodings = make(map[string]string) // uploadID -> encoding (gzip-stream-v1 | raw)
	timerLock       sync.Mutex
	encodingLock    sync.Mutex
	chunkDir        string
	UPLOAD_DIR      string
)

// getUploadEncoding returns the encoding associated with an upload session,
// if one has been established.
func getUploadEncoding(uploadID string) (string, bool) {
	encodingLock.Lock()
	defer encodingLock.Unlock()
	enc, ok := uploadEncodings[uploadID]
	return enc, ok
}

// setUploadEncoding records the encoding for an upload session.
func setUploadEncoding(uploadID, encoding string) {
	encodingLock.Lock()
	defer encodingLock.Unlock()
	uploadEncodings[uploadID] = encoding
}

func deleteUploadEncoding(uploadID string) {
	encodingLock.Lock()
	defer encodingLock.Unlock()
	delete(uploadEncodings, uploadID)
}

// resolveEncoding normalizes an incoming encoding value and validates it.
// An empty value falls back to gzip-stream-v1 for backward compatibility
// with older clients that always gzip-compressed chunks.
func resolveEncoding(raw string) (string, bool) {
	if raw == "" {
		return EncodingGzip, true
	}
	if raw == EncodingGzip || raw == EncodingRaw {
		return raw, true
	}
	return "", false
}

func handleChunkUpload(c *gin.Context) {
	uploadID := c.Param("uuid")
	chunkIndexStr := c.PostForm("chunkIndex")

	encoding, ok := resolveEncoding(c.PostForm("encoding"))
	if !ok {
		c.String(400, "Invalid encoding")
		return
	}

	// Associate the encoding with the upload session and validate that
	// subsequent chunks match the same mode. This prevents a raw chunk from
	// being mixed into a gzip session (or vice versa).
	if existing, found := getUploadEncoding(uploadID); found {
		if existing != encoding {
			c.String(400, "Encoding mismatch for upload session")
			return
		}
	} else {
		setUploadEncoding(uploadID, encoding)
	}

	file, _, err := c.Request.FormFile("chunk")
	if err != nil {
		c.String(400, "Missing chunk")
		return
	}
	defer file.Close()

	// Choose the reader based on the session encoding.
	//   - gzip-stream-v1: decompress the chunk before writing to disk.
	//   - raw: write the original bytes directly (no decompression, no base64).
	var reader io.Reader = file
	if encoding == EncodingGzip {
		gzReader, err := gzip.NewReader(file)
		if err != nil {
			c.String(400, "Could not decompress chunk. Make sure it's gzipped.")
			return
		}
		defer gzReader.Close()
		reader = gzReader
	}

	uploadPath := filepath.Join(chunkDir, uploadID)
	os.MkdirAll(uploadPath, os.ModePerm)

	chunkPath := filepath.Join(uploadPath, fmt.Sprintf("%s.part", chunkIndexStr))
	out, err := os.Create(chunkPath)
	if err != nil {
		c.String(500, "Failed to create chunk file")
		return
	}
	defer out.Close()
	io.Copy(out, reader)

	// Reset inactivity timer
	resetUploadTimer(uploadID)

	c.String(200, "Chunk received")
}

func finalizeUpload(c *gin.Context) {
	uploadID := c.Param("uuid")
	totalChunksStr := c.PostForm("totalChunks")
	originalFileName := c.PostForm("originalFileName")
	// md5sum := c.PostForm("md5sum") // No longer sent from frontend
	collectionID := c.PostForm("collectionId")

	userToken := c.PostForm("token")
	email := c.PostForm("email")
	password := c.PostForm("password")

	// Validate the encoding declared at finalization against the session.
	// This ensures the client cannot finalize a session under a different
	// mode than the one used to upload its chunks.
	encoding, ok := resolveEncoding(c.PostForm("encoding"))
	if !ok {
		c.String(400, "Invalid encoding")
		return
	}
	if existing, found := getUploadEncoding(uploadID); found && existing != encoding {
		c.String(400, "Encoding mismatch for upload session")
		return
	}

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

	uploadPath := filepath.Join(chunkDir, uploadID)
	missingChunks := checkMissingChunks(uploadPath, totalChunks)

	if len(missingChunks) > 0 {
		c.JSON(400, gin.H{"missingChunks": missingChunks, "message": "Some chunks are missing"})
		// Do not delete timer here, allow re-upload of missing chunks or timeout
		return
	}

	finalDestDir := filepath.Join(UPLOAD_DIR, "i")
	if err := os.MkdirAll(finalDestDir, os.ModePerm); err != nil {
		c.String(500, "Failed to create destination directory")
		return
	}
	// Create a temporary file first, we'll rename it after calculating the hash
	tempFile, err := os.CreateTemp(finalDestDir, "upload-*.tmp")
	if err != nil {
		c.String(500, "Failed to create temporary file")
		return
	}
	defer tempFile.Close()
	defer os.Remove(tempFile.Name()) // Ensure temp file is cleaned up on error

	// Assemble chunks in chunk-index order. By this point each stored .part
	// file already holds the original (decompressed) bytes: gzip chunks were
	// decompressed on receipt, raw chunks were stored as-is.
	if err := assembleChunkFiles(uploadPath, totalChunks, tempFile); err != nil {
		c.String(500, err.Error())
		return
	}

	// Calculate MD5 hash of the assembled file
	tempFile.Seek(0, 0) // Go back to the start of the file
	hash := md5.New()
	if _, err := io.Copy(hash, tempFile); err != nil {
		c.String(500, "Failed to calculate MD5 hash")
		return
	}
	md5sum := hex.EncodeToString(hash.Sum(nil))

	fileInfo, err := tempFile.Stat()
	if err != nil {
		c.String(500, "Failed to get final file stats")
		return
	}
	fileSize := fileInfo.Size()

	// Rename the temp file to its final name
	finalFilePath := filepath.Join(finalDestDir, md5sum+filepath.Ext(originalFileName))
	tempFile.Close() // Close the file before renaming
	if err := os.Rename(tempFile.Name(), finalFilePath); err != nil {
		c.String(500, "Failed to rename temporary file")
		return
	}

	uniqueFileName := database.GenerateUniqueFileName(originalFileName)
	// Insert file metadata into database
	fileData := database.FileData{
		OriginalFileName: originalFileName,
		FileDirectory:    uniqueFileName,
		AccountToken:     accountToken,
		FileSize:         fileSize,
		Timestamp:        time.Now().Unix(),
		Md5sum:           md5sum + filepath.Ext(originalFileName),
	}

	if err := fileData.Insert(); err != nil {
		os.Remove(finalFilePath) // Clean up if DB insert fails
		c.String(500, fmt.Sprintf("Failed to insert file metadata: %v", err))
		return
	}

	// If a collection ID is provided, add the file to the collection
	if collectionID != "" {
		addReq := socketHandler.AddFileToCollectionRequest{
			CollectionID:  collectionID,
			FileDirectory: fileData.FileDirectory,
			Auth:          socketHandler.AuthInfo{Token: accountToken},
		}
		if _, err := socketHandler.AddFileToCollection(addReq); err != nil {
			// Log this error, but don't fail the entire upload.
			// The file is uploaded, just not added to the collection.
			fmt.Printf("Warning: Failed to add file %s to collection %s: %v\n", fileData.FileDirectory, collectionID, err)
		}
	}

	// Clean up: Stop timer and remove chunk directory
	timerLock.Lock()
	if timer, ok := uploadTimers[uploadID]; ok {
		timer.Stop()
		delete(uploadTimers, uploadID)
	}
	timerLock.Unlock()
	deleteUploadEncoding(uploadID)

	if err := os.RemoveAll(uploadPath); err != nil {
		// Log this error, but the upload is mostly successful
		fmt.Printf("Warning: Failed to remove chunk directory %s: %v\n", uploadPath, err)
	}

	var FileUpdate socketHandler.FileUpdate
	FileUpdate.File = fileData
	FileUpdate.Toggle = true
	go socketHandler.UserFilesPulse(FileUpdate)
	go socketHandler.UpdateUserCount()

	c.JSON(200, gin.H{
		"message":       "Upload successful and file assembled",
		"fileName":      uniqueFileName,
		"fileDirectory": uniqueFileName, // Consistent with FileData
		"accessPath":    fmt.Sprintf("/i/%s", uniqueFileName),
	})
}

// assembleChunkFiles reads .part files from uploadPath in chunk-index order
// and writes them sequentially into dest. Each .part file already contains
// the original (decompressed) bytes regardless of the upload encoding.
func assembleChunkFiles(uploadPath string, totalChunks int, dest io.Writer) error {
	for i := 0; i < totalChunks; i++ {
		chunkPath := filepath.Join(uploadPath, fmt.Sprintf("%d.part", i))
		chunkFile, err := os.Open(chunkPath)
		if err != nil {
			return fmt.Errorf("failed to open chunk %d: %v", i, err)
		}
		_, err = io.Copy(dest, chunkFile)
		chunkFile.Close()
		if err != nil {
			return fmt.Errorf("failed to copy chunk %d: %v", i, err)
		}
	}
	return nil
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
			deleteUploadEncoding(uploadID)
			fmt.Printf("Upload %s expired and deleted\n", uploadID)
		})
	}
}

func setupUploaderRoutes(r *gin.Engine, UPLOAD_DIR_BASE string) {
	chunkDir = UPLOAD_DIR_BASE + "/tmp_chunks"
	UPLOAD_DIR = UPLOAD_DIR_BASE
	os.RemoveAll(chunkDir)
	os.MkdirAll(chunkDir, os.ModePerm)
	r.POST("/upload/:uuid", handleChunkUpload)
	r.POST("/upload/success/:uuid", finalizeUpload)
}
