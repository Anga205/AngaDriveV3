package endpoints

import (
	"angadrive/config"
	"angadrive/service"
	"angadrive/storage"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const uploadTimeout = 5 * time.Minute

var (
	uploadTimers = make(map[string]*time.Timer)
	timerLock    sync.Mutex
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

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		c.String(400, "Could not decompress chunk. Make sure it's gzipped.")
		return
	}
	defer gzReader.Close()

	uploadPath := filepath.Join(storage.ChunksDir(), uploadID)
	os.MkdirAll(uploadPath, os.ModePerm)

	chunkPath := filepath.Join(uploadPath, fmt.Sprintf("%s.part", chunkIndexStr))
	out, err := os.Create(chunkPath)
	if err != nil {
		c.String(500, "Failed to create chunk file")
		return
	}
	defer out.Close()
	io.Copy(out, gzReader)

	resetUploadTimer(uploadID)
	c.String(200, "Chunk received")
}

func finalizeUpload(c *gin.Context) {
	uploadID := c.Param("uuid")
	totalChunksStr := c.PostForm("totalChunks")
	originalFileName := c.PostForm("originalFileName")
	collectionID := c.PostForm("collectionId")

	userToken := c.PostForm("token")
	email := c.PostForm("email")
	password := c.PostForm("password")

	if originalFileName == "" {
		c.String(400, "Missing originalFileName")
		return
	}

	var accountToken string
	var err error
	if email != "" && password != "" {
		accountToken, err = service.AuthenticateEmailPassword(email, password)
		if err != nil {
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

	fileData, err := service.FinalizeUpload(accountToken, uploadID, originalFileName, collectionID, totalChunks)
	if err != nil {
		if missing, ok := err.(*service.MissingChunksError); ok {
			c.JSON(400, gin.H{"missingChunks": missing.Missing, "message": "Some chunks are missing"})
			return
		}
		c.String(500, err.Error())
		return
	}

	stopUploadTimer(uploadID)

	c.JSON(200, gin.H{
		"message":       "Upload successful and file assembled",
		"fileName":      fileData.FileDirectory,
		"fileDirectory": fileData.FileDirectory,
		"accessPath":    fmt.Sprintf("/i/%s", fileData.FileDirectory),
	})
}

func resetUploadTimer(uploadID string) {
	timerLock.Lock()
	defer timerLock.Unlock()

	if timer, ok := uploadTimers[uploadID]; ok {
		timer.Reset(uploadTimeout)
		return
	}
	uploadTimers[uploadID] = time.AfterFunc(uploadTimeout, func() {
		os.RemoveAll(filepath.Join(storage.ChunksDir(), uploadID))
		timerLock.Lock()
		delete(uploadTimers, uploadID)
		timerLock.Unlock()
		fmt.Printf("Upload %s expired and deleted\n", uploadID)
	})
}

func stopUploadTimer(uploadID string) {
	timerLock.Lock()
	defer timerLock.Unlock()
	if timer, ok := uploadTimers[uploadID]; ok {
		timer.Stop()
		delete(uploadTimers, uploadID)
	}
}

func setupUploaderRoutes(r *gin.Engine) {
	os.MkdirAll(storage.ChunksDir(), os.ModePerm)
	guard := func(h gin.HandlerFunc) gin.HandlerFunc {
		return func(c *gin.Context) {
			if c.Request.Host != config.AssetsURL && c.Request.Host != config.WebURL {
				c.AbortWithStatus(404)
				return
			}
			h(c)
		}
	}
	r.POST("/upload/:uuid", guard(handleChunkUpload))
	r.POST("/upload/success/:uuid", guard(finalizeUpload))
}
