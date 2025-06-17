package accounts

import (
	"service/database"
	"sync"

	"golang.org/x/crypto/bcrypt"
)

var (
	bcryptCache      = make(map[string]bool)
	bcryptCacheMutex sync.RWMutex
	bcryptCacheOrder []string
	bcryptCacheSize  = 512 // LRU size
)

func cacheBcryptResult(key string, result bool) {
	bcryptCacheMutex.Lock()
	defer bcryptCacheMutex.Unlock()
	if _, exists := bcryptCache[key]; !exists && len(bcryptCacheOrder) >= bcryptCacheSize {
		// Remove oldest
		oldest := bcryptCacheOrder[0]
		bcryptCacheOrder = bcryptCacheOrder[1:]
		delete(bcryptCache, oldest)
	}
	if _, exists := bcryptCache[key]; !exists {
		bcryptCacheOrder = append(bcryptCacheOrder, key)
	}
	bcryptCache[key] = result
}

func getBcryptCache(key string) (bool, bool) {
	bcryptCacheMutex.RLock()
	defer bcryptCacheMutex.RUnlock()
	val, ok := bcryptCache[key]
	return val, ok
}

func Authenticate(email string, password string) bool {
	user, err := database.FindUserByEmail(email)
	if err != nil {
		return false
	}

	cacheKey := user.HashedPassword + ":" + password
	if result, ok := getBcryptCache(cacheKey); ok {
		return result
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(password))
	result := err == nil
	cacheBcryptResult(cacheKey, result)
	return result
}

func AuthenticateHashed(email string, hashedPassword string) bool {
	user, err := database.FindUserByEmail(email)
	if err != nil {
		return false
	}

	if email == user.Email && hashedPassword == user.HashedPassword {
		return true
	}
	return false
}
