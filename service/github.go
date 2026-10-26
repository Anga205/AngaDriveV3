package service

import (
	"angadrive/database"
	"angadrive/hub"
	"angadrive/protocol"
	"angadrive/storage"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/net/context"
)

var githubRepoPattern = regexp.MustCompile(`^https://github\.com/[a-zA-Z0-9_-]+/[a-zA-Z0-9_-]+$`)

// GithubImportHandler clones a GitHub repository and imports its files into a
// new collection tree.
func GithubImportHandler(req protocol.ImportGithubRepoRequest) (string, error) {
	if !githubRepoPattern.MatchString(req.RepoURL) {
		return "", errors.New("invalid github url")
	}

	userToken, err := Authenticate(req.Auth)
	if err != nil {
		return "", err
	}
	parts := strings.Split(req.RepoURL, "/")
	repoName := parts[len(parts)-1]
	repoOwner := parts[len(parts)-2]

	// Check repository size via GitHub API.
	resp, err := http.Get(fmt.Sprintf("https://api.github.com/repos/%s/%s", repoOwner, repoName))
	if err != nil {
		return "", fmt.Errorf("failed to fetch repository details: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch repository details: received status code %d", resp.StatusCode)
	}
	var repoDetails struct {
		Size int `json:"size"` // kilobytes
	}
	if err := json.NewDecoder(resp.Body).Decode(&repoDetails); err != nil {
		return "", fmt.Errorf("failed to parse repository details: %w", err)
	}
	if repoDetails.Size > 2621440 { // 2.5 GB
		return "", errors.New("repository size exceeds the 2.5GB limit")
	}

	hub.NotifyUser(userToken, map[string]interface{}{"type": "notification", "data": "Starting Import...."})

	clonedUUID := uuid.New().String()
	folderToCloneTo := filepath.Join(storage.ReposDir(), clonedUUID)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", req.RepoURL, folderToCloneTo)
	if err := cmd.Run(); err != nil {
		os.RemoveAll(folderToCloneTo)
		if ctx.Err() != nil {
			return "", errors.New("git clone operation timed out")
		}
		return "", err
	}

	rootCollection, err := database.InsertNewCollection(database.Collection{
		Name:      repoName,
		Editors:   userToken,
		Size:      0,
		Timestamp: time.Now().Unix(),
	})
	if err != nil {
		os.RemoveAll(folderToCloneTo)
		return "", fmt.Errorf("failed to create root collection: %w", err)
	}
	collectionMap := map[string]database.Collection{clonedUUID: rootCollection}

	walkErr := filepath.WalkDir(folderToCloneTo, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == ".git" {
			return os.RemoveAll(path)
		}
		if len(strings.Split(path, string(os.PathSeparator))) <= 3 {
			return nil // skip the root and shallow entries
		}
		dirStructure := strings.Split(path, string(os.PathSeparator))[2:]
		parentKey := strings.Join(dirStructure[:len(dirStructure)-1], string(os.PathListSeparator))
		parentDir, ok := collectionMap[parentKey]
		if !ok {
			return nil
		}

		if !d.IsDir() {
			fileSHA256, err := storage.SHA256File(path)
			if err != nil {
				return err
			}
			info, err := d.Info()
			if err != nil {
				return err
			}
			ext := filepath.Ext(d.Name())
			newFile := database.FileData{
				OriginalFileName: d.Name(),
				FileDirectory:    database.GenerateUniqueFileName(d.Name()),
				AccountToken:     userToken,
				Timestamp:        time.Now().Unix(),
				Sha256sum:        fileSHA256 + ext,
				FileSize:         info.Size(),
			}
			if err := newFile.Insert(); err != nil {
				return err
			}
			if err := parentDir.AddFile(newFile.FileDirectory); err != nil {
				return err
			}
			if err := os.Rename(path, storage.FilePath(fileSHA256+ext)); err != nil {
				return err
			}
			return nil
		}

		newCollection, err := database.InsertNewCollection(database.Collection{
			Name:      d.Name(),
			Editors:   userToken,
			Size:      0,
			Timestamp: time.Now().Unix(),
			Dependant: clonedUUID,
		})
		if err != nil {
			return err
		}
		if err := parentDir.AddFolder(newCollection.ID); err != nil {
			return err
		}
		collectionMap[strings.Join(dirStructure, string(os.PathListSeparator))] = newCollection
		return nil
	})
	os.RemoveAll(folderToCloneTo)
	if walkErr != nil {
		return "", walkErr
	}

	if updated, err := database.GetCollection(rootCollection.ID); err == nil {
		go hub.PublishCollectionCard(true, updated)
	}
	go hub.PublishFileCount()
	hub.NotifyUser(userToken, map[string]interface{}{"type": "notification", "data": "Import Completed Successfully!"})
	return folderToCloneTo, nil
}
