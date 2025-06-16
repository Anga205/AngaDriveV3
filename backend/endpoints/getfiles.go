package endpoints

import (
	"errors"
	"net/http"
	"service/accounts"
	"service/database"

	"github.com/gin-gonic/gin"
)

type AuthRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Token    string `json:"token"`
}

func getUserFilesHTTP(c *gin.Context) {
	var req AuthRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	files, err := GetUserFiles(req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, files)
}

func GetUserFiles(req AuthRequest) ([]database.FileData, error) {

	if req.Token == "" && (req.Email == "" || req.Password == "") {
		return nil, errors.New("missing authentication credentials")
	}

	if req.Token == "" {
		if !accounts.Authenticate(req.Email, req.Password) {
			return nil, errors.New("invalid email or password")
		}
		userInfo, _ := database.FindUserByEmail(req.Email)
		req.Token = userInfo.Token
	}
	files, err := database.GetUserFiles(req.Token)
	if err != nil {
		return nil, errors.New("failed to retrieve files")
	}
	return files, nil

}
