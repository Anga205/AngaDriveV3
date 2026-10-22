package accounts

import (
	"angadrive/database"
	"fmt"
)

func LoginUser(request LoginRequest) (database.Account, error) {
	user, err := database.FindUserByEmail(request.Email)
	if err != nil {
		// Do not reveal whether the email exists.
		return database.Account{}, fmt.Errorf("invalid credentials")
	}

	if request.Legacy {
		// The client still holds the plaintext password for a pre-migration
		// account. Verify it and upgrade the stored hash in place.
		if !AuthenticateLegacy(request.Email, request.Password) {
			return database.Account{}, fmt.Errorf("invalid credentials")
		}
		// Re-read so the response reflects the upgraded hash.
		user, err = database.FindUserByEmail(request.Email)
		if err != nil {
			return database.Account{}, err
		}
		return user, nil
	}

	if user.PasswordVersion == 0 {
		// The account has not been migrated yet and the client sent a digest.
		// Ask the client to retry with the plaintext so we can upgrade it.
		return database.Account{}, ErrLegacyPassword
	}

	if !Authenticate(request.Email, request.Password) {
		return database.Account{}, fmt.Errorf("invalid credentials")
	}
	return user, nil
}
