package accounts

import (
	"angadrive/database"
	"fmt"
)

// MigratePassword upgrades a legacy account from bcrypt(plaintext) to
// bcrypt(sha256(plaintext)) without requiring the user to log in again.
//
// The frontend calls this when it finds a plaintext password cached in
// localStorage from before the migration. OldPassword is that plaintext and
// NewPasswordHashed is bcrypt(sha256(OldPassword)), computed in the browser.
//
// The operation is idempotent: if the account was already migrated (for example
// by another tab), the plaintext is hashed before comparison and the stored
// hash is left untouched.
func MigratePassword(request MigratePasswordRequest) (database.Account, error) {
	user, err := database.FindUserByEmail(request.Email)
	if err != nil {
		return database.Account{}, err
	}

	if user.PasswordVersion >= 1 {
		// Already migrated. Accept the request only if the plaintext still
		// matches, so a stale client cannot overwrite the hash.
		if !verifySecret(user, sha256Hex(request.OldPassword)) {
			return database.Account{}, fmt.Errorf("invalid credentials")
		}
		return user, nil
	}

	if !verifySecret(user, request.OldPassword) {
		return database.Account{}, fmt.Errorf("invalid credentials")
	}

	newUser := user
	newUser.HashedPassword = request.NewPasswordHashed
	newUser.PasswordVersion = 1
	if err := user.Update(newUser); err != nil {
		return database.Account{}, fmt.Errorf("failed to migrate password: %v", err)
	}
	return newUser, nil
}
