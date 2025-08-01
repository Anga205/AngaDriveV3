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

	"service/accounts"
	"service/database"
	"service/socketHandler"

	"github.com/gin-gonic/gin"
)

const timeout = 5 * time.Minute

var (
	uploadTimers = make(map[string]*time.Timer)
	timerLock    sync.Mutex
	chunkDir     string
	UPLOAD_DIR   string
)

func handleChunkUpload(c *gin.Context) {
	uploadID := c.Param("uuid")
	chunkIndexStr := c.PostForm("chunkIndex")

	file, _, err := c.Request.FormFile("chunk")
	if err != nil {
		c.String(400, "Missing chunk")
		return
	}
	defer file.Close()

	// Decompress gzipped chunk
	gzReader, err := gzip.NewReader(file)
	if err != nil {
		c.String(400, "Could not decompress chunk. Make sure it's gzipped.")
		return
	}
	defer gzReader.Close()

	uploadPath := filepath.Join(chunkDir, uploadID)
	os.MkdirAll(uploadPath, os.ModePerm)

	chunkPath := filepath.Join(uploadPath, fmt.Sprintf("%s.part", chunkIndexStr))
	out, err := os.Create(chunkPath)
	if err != nil {
		c.String(500, "Failed to create chunk file")
		return
	}
	defer out.Close()
	io.Copy(out, gzReader)

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

	for i := 0; i < totalChunks; i++ {
		chunkPath := filepath.Join(uploadPath, fmt.Sprintf("%d.part", i))
		chunkFile, err := os.Open(chunkPath)
		if err != nil {
			c.String(500, fmt.Sprintf("Failed to open chunk %d: %v", i, err))
			return
		}
		_, err = io.Copy(tempFile, chunkFile)
		chunkFile.Close() // Close chunk file immediately after copy
		if err != nil {
			c.String(500, fmt.Sprintf("Failed to copy chunk %d: %v", i, err))
			return
		}
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
	r.POST("/upload/:uuid", handleChunkUpload)
	r.POST("/upload/success/:uuid", finalizeUpload)
}
