package socketHandler

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"service/database"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/net/context"
)

func normalizeToJSON(data map[string]interface{}) string {
	// Convert map to JSON with 4-space indentation
	jsonBytes, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		log.Fatalf("JSON marshaling failed: %v", err)
	}
	return string(jsonBytes)
}

func convertToInterfaceMap(collectionMap map[string]database.Collection) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range collectionMap {
		result[k] = v
	}
	return result
}

func FileMD5(filePath string) string {
	file, _ := os.Open(filePath)
	defer file.Close()
	hash := md5.New()
	io.Copy(hash, file)
	checksum := hash.Sum(nil)
	return fmt.Sprintf("%x", checksum)
}

func GithubImportHandler(req ImportGithubRepoRequest) (string, error) {

	matched, err := regexp.MatchString(`^https:\/\/github\.com\/[a-zA-Z0-9_-]+\/[a-zA-Z0-9_-]+$`, req.RepoURL)
	if err != nil {
		return "", err
	}

	if !matched {
		return "", errors.New("invalid github url")
	}

	userToken, err := req.Auth.GetToken()
	if err != nil {
		return "", err
	}
	repoBreakDown := strings.Split(req.RepoURL, "/")
	repoName := repoBreakDown[len(repoBreakDown)-1]
	repoOwner := repoBreakDown[len(repoBreakDown)-2]

	// Check repository size via GitHub API
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s", repoOwner, repoName)
	resp, err := http.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch repository details: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch repository details: received status code %d", resp.StatusCode)
	}

	var repoDetails struct {
		Size int `json:"size"` // Size in kilobytes
	}

	if err := json.NewDecoder(resp.Body).Decode(&repoDetails); err != nil {
		return "", fmt.Errorf("failed to parse repository details: %w", err)
	}

	// 2.5 GB in KB = 2.5 * 1024 * 1024 = 2621440
	if repoDetails.Size > 2621440 {
		return "", errors.New("repository size exceeds the 2.5GB limit")
	}

	genericUserPulse(userToken, map[string]interface{}{
		"type": "notification",
		"data": "Starting Import....",
	})
	clonedUUID := uuid.New().String()
	folderToCloneTo := filepath.Join(UPLOAD_DIR, "repos", clonedUUID)
	// this creates a context with a timeout of 15 minutes, so if a git import takes longer than that, it will be cancelled
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", req.RepoURL, folderToCloneTo)
	err = cmd.Run()
	result := make(chan error, 1)

	go func() {
		result <- err
	}()

	rootCollection := database.Collection{
		Name:      repoName,
		Editors:   userToken,
		Size:      0,
		Timestamp: time.Now().Unix(),
	}
	rootCollection.Insert()
	collectionMap := map[string]database.Collection{
		clonedUUID: rootCollection,
	}
	select {
	case err := <-result:
		if err != nil {
			// Clean up the folder if the clone fails
			os.RemoveAll(folderToCloneTo)
			return "", err
		}
		err = filepath.WalkDir(folderToCloneTo, func(path string, d os.DirEntry, err error) error {
			if d.IsDir() && d.Name() == ".git" {
				os.RemoveAll(path)
			} else {
				if len(strings.Split(path, string(os.PathSeparator))) <= 3 {
					return nil // Skip the root directory and any directories that are not part of the cloned repo
				}
				fmt.Println("\n\n\n\nWalking through:\n", normalizeToJSON(convertToInterfaceMap(collectionMap)))
				dirStructure := strings.Split(path, string(os.PathSeparator))[2:]
				parentDir := collectionMap[strings.Join(dirStructure[:len(dirStructure)-1], string(os.PathListSeparator))]
				if !d.IsDir() {
					fileMD5 := FileMD5(path)
					newFileName := d.Name()
					fileDir := database.GenerateUniqueFileName(newFileName)
					fileInfo, _ := d.Info()
					fileExtension := filepath.Ext(newFileName)
					newFile := database.FileData{
						OriginalFileName: newFileName,
						FileDirectory:    fileDir,
						AccountToken:     userToken,
						Timestamp:        time.Now().Unix(),
						Md5sum:           fileMD5 + fileExtension,
						FileSize:         fileInfo.Size(),
					}
					newFile.Insert()
					parentDir.AddFile(newFile.FileDirectory)
					newPath := filepath.Join(UPLOAD_DIR, "i", fileMD5+fileExtension)
					err := os.Rename(path, newPath)
					if err != nil {
						fmt.Println("Error moving file:", err)
						return err
					}
				} else { // Avoid creating a duplicate collection for the root cloned directory
					newCollection := database.Collection{
						Name:        d.Name(),
						Editors:     userToken,
						Size:        0,
						Collections: "",
						Files:       "",
						Timestamp:   time.Now().Unix(),
						Dependant:   clonedUUID,
					}
					newCollection.Insert()
					parentDir.AddFolder(newCollection.ID)
					collectionMap[strings.Join(dirStructure, string(os.PathListSeparator))] = newCollection
				}
				collectionMap[strings.Join(dirStructure[:len(dirStructure)-1], string(os.PathListSeparator))] = parentDir
			}
			return nil
		})
		os.RemoveAll(folderToCloneTo)
		if err != nil {
			return "", err
		}
		collectionUpdate, _ := database.GetCollection(rootCollection.ID)
		go CollectionPulse(true, collectionUpdate)
		return folderToCloneTo, nil
	case <-ctx.Done():
		// If the context is done, it means the operation timed out
		os.RemoveAll(folderToCloneTo)
		return "", errors.New("git clone operation timed out")
	}

}
