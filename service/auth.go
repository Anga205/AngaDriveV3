// Package service implements AngaDrive's business operations. It sits between
// the transports (websocket/HTTP) and the lower layers (database, storage,
// runners, hub). Transports parse requests and call these functions; these
// functions never touch gin or websocket types.
package service

import (
	"angadrive/accounts"
	"angadrive/database"
	"angadrive/protocol"
	"fmt"
)

// Authenticate resolves request credentials to an account token. It is the
// single implementation used by every transport.
func Authenticate(auth protocol.AuthInfo) (string, error) {
	return auth.GetToken()
}

// AuthenticateOrToken returns auth.Token if set, otherwise resolves
// email/password credentials.
func AuthenticateOrToken(auth protocol.AuthInfo) (string, error) {
	if auth.Token != "" {
		return auth.Token, nil
	}
	return Authenticate(auth)
}

// AuthenticateEmailPassword verifies email/password credentials and returns the
// account token.
func AuthenticateEmailPassword(email, password string) (string, error) {
	if !accounts.Authenticate(email, password) {
		return "", fmt.Errorf("invalid email or password")
	}
	user, err := database.FindUserByEmail(email)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve user details")
	}
	return user.Token, nil
}

// AccountForCredentials verifies email/password credentials and returns the
// matching account.
func AccountForCredentials(email, password string) (database.Account, error) {
	if !accounts.Authenticate(email, password) {
		return database.Account{}, fmt.Errorf("invalid email or password")
	}
	return database.FindUserByEmail(email)
}
