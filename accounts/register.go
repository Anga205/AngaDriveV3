package accounts

import (
	"angadrive/database"
	"crypto/rand"
	"fmt"
	"strings"
	"time"
)

func GenToken() string { // Sample Output: "VcW90bRMW8.HLrITln50boVlu5Sw3eI.1747739198"
	letters := "qwertyuiopasdfghjklzxcvbnm"
	letters += strings.ToUpper(letters)
	letters += "1234567890"

	part1 := randomString(10, letters)
	part2 := randomString(20, letters)

	timestamp := time.Now().Unix()
	return part1 + "." + part2 + "." + fmt.Sprintf("%d", timestamp)
}

// randomString returns n characters drawn uniformly from alphabet using
// rejection sampling to avoid modulo bias.
func randomString(n int, alphabet string) string {
	out := make([]byte, n)
	buf := make([]byte, 1)
	limit := byte(256 - (256 % len(alphabet)))
	for i := 0; i < n; {
		if _, err := rand.Read(buf); err != nil {
			fmt.Println("Error generating random byte in accounts.register.GenToken:", err)
			panic(err)
		}
		if buf[0] >= limit {
			continue // reject to keep the distribution uniform
		}
		out[i] = alphabet[int(buf[0])%len(alphabet)]
		i++
	}
	return string(out)
}

func RegisterUser(RequestInfo RegisterRequest) (database.Account, error) {
	NewUser := database.Account{
		Token:          GenToken(),
		DisplayName:    RequestInfo.DisplayName,
		Email:          RequestInfo.Email,
		HashedPassword: RequestInfo.HashedPassword,
		// New accounts always use bcrypt(sha256(password)).
		PasswordVersion: 1,
	}
	// Rely on the unique index on email to close the check-then-insert race.
	if err := NewUser.Insert(); err != nil {
		if database.IsDuplicateKeyError(err) {
			return database.Account{}, fmt.Errorf("email already exists")
		}
		return database.Account{}, fmt.Errorf("failed to insert new user: %v", err)
	}

	return NewUser, nil
}
