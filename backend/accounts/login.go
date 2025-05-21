package accounts

import (
	"fmt"
	"service/database"
)

func LoginUser(request LoginRequest) (database.Account, error) {
	user, err := database.FindUserByEmail(request.Email)
	if err != nil {
		return database.Account{}, err
	}

	if Authenticate(UserInfo(request)) {
		return user, nil
	}

	return database.Account{}, fmt.Errorf("invalid credentials")
}
