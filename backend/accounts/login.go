package accounts

import (
	"fmt"
	"service/database"
)

func LoginUser(request LoginRequest) (database.Account, error) {

	if Authenticate(request.Email, request.Password) {
		user, err := database.FindUserByEmail(request.Email)
		if err != nil {
			return database.Account{}, err
		}
		return user, nil
	}

	return database.Account{}, fmt.Errorf("invalid credentials")
}
