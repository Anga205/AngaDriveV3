package accounts

import (
	"crypto/rand"
	"fmt"
	"service/database"
	"strings"
	"time"
)

func GenToken() string { // Sample Output: "VcW90bRMW8.HLrITln50boVlu5Sw3eI.1747739198"
	letters := "qwertyuiopasdfghjklzxcvbnm"
	letters += strings.ToUpper(letters)
	letters += "1234567890"

	part1 := make([]byte, 10)
	part2 := make([]byte, 20)
	for i := range part1 {
		b := make([]byte, 1)
		_, err := rand.Read(b)
		if err != nil {
			fmt.Println("Error generating random byte in accounts.register.GenToken:", err)
			panic(err)
		}
		part1[i] = letters[int(b[0])%len(letters)]
	}
	for i := range part2 {
		b := make([]byte, 1)
		_, err := rand.Read(b)
		if err != nil {
			fmt.Println("Error generating random byte in accounts.register.GenToken:", err)
			panic(err)
		}
		part2[i] = letters[int(b[0])%len(letters)]
	}

	timestamp := time.Now().Unix()
	return string(part1) + "." + string(part2) + "." + fmt.Sprintf("%d", timestamp)
}

func RegisterUser(RequestInfo RegisterRequest) (RegisterResponse, error) {
	_, err := database.FindUserByEmail(RequestInfo.Email)
	if err == nil {
		return RegisterResponse{}, fmt.Errorf("email already exists")
	}

	NewUser := database.Account{
		Token:          GenToken(),
		DisplayName:    RequestInfo.DisplayName,
		Email:          RequestInfo.Email,
		HashedPassword: RequestInfo.HashedPassword,
	}
	db := database.GetDB()
	err = db.Create(&NewUser).Error
	if err != nil {
		return RegisterResponse{}, err
	}

	go func() {
		database.UserAccountsMutex.Lock()
		database.UserAccounts[NewUser.Email] = NewUser
		database.UserAccountsMutex.Unlock()
	}()
	return RegisterResponse{NewUser.Token}, nil
}
