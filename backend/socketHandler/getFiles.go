package socketHandler

import (
	"errors"
	"service/accounts"
	"service/database"
)

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
