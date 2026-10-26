package accounts

import (
	"angadrive/database"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"sync"

	"golang.org/x/crypto/bcrypt"
)

// ErrLegacyPassword is returned by LoginUser when the account still uses the
// pre-migration scheme (bcrypt of the plaintext password) and the client sent a
// SHA-256 digest. The client should retry with Legacy=true and the plaintext.
var ErrLegacyPassword = errors.New("legacy_password")

// ErrStaleClient is returned by LoginUser when the client sent a plaintext
// password without Legacy=true. The current frontend always sends a SHA-256
// digest, so this means the browser is running a cached pre-migration build
// that has not picked up the new frontend yet. The user must hard-refresh.
var ErrStaleClient = errors.New("stale_client")

// StaleClientMessage is shown to users whose browser is running an outdated
// cached frontend that still sends plaintext passwords.
const StaleClientMessage = "Your browser is running an outdated version of AngaDrive. " +
	"Please hard-refresh the page (Ctrl+Shift+R, or Cmd+Shift+R on macOS) to continue."

// isSHA256Hex reports whether s looks like a lowercase hex SHA-256 digest.
func isSHA256Hex(s string) bool {
	if len(s) != 64 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

// sha256Hex returns the lowercase hex SHA-256 digest of s. This is the value the
// frontend sends in place of the plaintext password.
func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

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

// Authenticate reports whether secret matches the account's stored bcrypt hash.
// For migrated accounts secret is the hex SHA-256 digest of the password; for
// legacy accounts it is the plaintext password.
func Authenticate(email string, secret string) bool {
	user, err := database.FindUserByEmail(email)
	if err != nil {
		return false
	}
	return verifySecret(user, secret)
}

// verifySecret compares secret against the account's stored bcrypt hash,
// memoizing the result in the LRU cache. The cache key is a hash of the secret
// so the plaintext/digest is never retained in memory as a map key.
func verifySecret(user database.Account, secret string) bool {
	cacheKey := user.HashedPassword + ":" + sha256Hex(secret)
	if result, ok := getBcryptCache(cacheKey); ok {
		return result
	}

	err := bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(secret))
	result := err == nil
	cacheBcryptResult(cacheKey, result)
	return result
}

// AuthenticateLegacy verifies a plaintext password against a pre-migration
// account (bcrypt of the plaintext) and, on success, upgrades the stored hash to
// bcrypt(sha256(plaintext)) so the account joins the current scheme.
//
// If the account has already been migrated, the plaintext is hashed first so a
// stale client that still holds the plaintext can still log in.
func AuthenticateLegacy(email string, plaintext string) bool {
	user, err := database.FindUserByEmail(email)
	if err != nil {
		return false
	}

	if user.PasswordVersion >= 1 {
		return verifySecret(user, sha256Hex(plaintext))
	}

	if !verifySecret(user, plaintext) {
		return false
	}

	// Verification succeeded. A failure to persist the upgrade must not block
	// the login; the next login will retry the upgrade.
	_ = upgradePassword(user, sha256Hex(plaintext))
	return true
}

// upgradePassword stores bcrypt(secret) for the account and marks it migrated.
func upgradePassword(user database.Account, secret string) error {
	hashed, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	newUser := user
	newUser.HashedPassword = string(hashed)
	newUser.PasswordVersion = 1
	return user.Update(newUser)
}

func AuthenticateHashed(email string, hashedPassword string) bool {
	user, err := database.FindUserByEmail(email)
	if err != nil {
		return false
	}
	// Constant-time comparison to avoid leaking the stored hash via timing.
	return subtle.ConstantTimeCompare([]byte(hashedPassword), []byte(user.HashedPassword)) == 1
}
