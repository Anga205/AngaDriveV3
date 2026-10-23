package accounts

import (
	"angadrive/database"
	"os"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// setupTestDB initializes a fresh SQLite database in a temporary directory and
// restores the working directory when the test finishes.
func setupTestDB(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	if err := database.InitializeDatabase(); err != nil {
		t.Fatalf("InitializeDatabase: %v", err)
	}
}

// insertLegacyAccount stores an account using the pre-migration scheme:
// bcrypt(plaintext) with PasswordVersion 0.
func insertLegacyAccount(t *testing.T, email, plaintext string) database.Account {
	t.Helper()
	hashed, err := bcrypt.GenerateFromPassword([]byte(plaintext), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	account := database.Account{
		Token:           GenToken(),
		DisplayName:     "Legacy",
		Email:           email,
		HashedPassword:  string(hashed),
		PasswordVersion: 0,
	}
	if err := account.Insert(); err != nil {
		t.Fatalf("insert: %v", err)
	}
	return account
}

func TestAuthenticateLegacyUpgradesAccount(t *testing.T) {
	setupTestDB(t)
	const email = "legacy@example.com"
	const plaintext = "hunter2"
	insertLegacyAccount(t, email, plaintext)

	if !AuthenticateLegacy(email, plaintext) {
		t.Fatal("AuthenticateLegacy rejected the correct plaintext")
	}

	user, err := database.FindUserByEmail(email)
	if err != nil {
		t.Fatalf("FindUserByEmail: %v", err)
	}
	if user.PasswordVersion != 1 {
		t.Fatalf("PasswordVersion = %d, want 1", user.PasswordVersion)
	}
	// The stored hash must now be bcrypt(sha256(plaintext)).
	if err := bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(sha256Hex(plaintext))); err != nil {
		t.Fatalf("stored hash is not bcrypt(sha256(plaintext)): %v", err)
	}
	// The digest must authenticate through the normal path.
	if !Authenticate(email, sha256Hex(plaintext)) {
		t.Fatal("Authenticate rejected the digest after migration")
	}
	// A stale client still holding the plaintext must keep working.
	if !AuthenticateLegacy(email, plaintext) {
		t.Fatal("AuthenticateLegacy rejected the plaintext after migration")
	}
}

func TestAuthenticateLegacyRejectsWrongPassword(t *testing.T) {
	setupTestDB(t)
	const email = "legacy2@example.com"
	insertLegacyAccount(t, email, "correct-horse")

	if AuthenticateLegacy(email, "wrong-password") {
		t.Fatal("AuthenticateLegacy accepted a wrong plaintext")
	}
	user, _ := database.FindUserByEmail(email)
	if user.PasswordVersion != 0 {
		t.Fatalf("PasswordVersion = %d, want 0 after a failed attempt", user.PasswordVersion)
	}
}

func TestMigratePassword(t *testing.T) {
	setupTestDB(t)
	const email = "migrate@example.com"
	const plaintext = "migrate-me"
	insertLegacyAccount(t, email, plaintext)

	newHash, err := bcrypt.GenerateFromPassword([]byte(sha256Hex(plaintext)), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}

	account, err := MigratePassword(MigratePasswordRequest{
		Email:             email,
		OldPassword:       plaintext,
		NewPasswordHashed: string(newHash),
	})
	if err != nil {
		t.Fatalf("MigratePassword: %v", err)
	}
	if account.PasswordVersion != 1 {
		t.Fatalf("PasswordVersion = %d, want 1", account.PasswordVersion)
	}
	if !Authenticate(email, sha256Hex(plaintext)) {
		t.Fatal("digest login failed after MigratePassword")
	}

	// A stale plaintext must not be able to overwrite the migrated hash.
	attackerHash, _ := bcrypt.GenerateFromPassword([]byte(sha256Hex("attacker")), bcrypt.DefaultCost)
	if _, err := MigratePassword(MigratePasswordRequest{
		Email:             email,
		OldPassword:       "not-the-password",
		NewPasswordHashed: string(attackerHash),
	}); err == nil {
		t.Fatal("MigratePassword accepted a wrong plaintext")
	}
	if !Authenticate(email, sha256Hex(plaintext)) {
		t.Fatal("migrated hash was overwritten by a rejected request")
	}
}

func TestLoginUserLegacyFlow(t *testing.T) {
	setupTestDB(t)
	const email = "login-legacy@example.com"
	const plaintext = "login-pass"
	insertLegacyAccount(t, email, plaintext)

	// A digest login on a legacy account must ask the client to retry.
	if _, err := LoginUser(LoginRequest{Email: email, Password: sha256Hex(plaintext)}); err != ErrLegacyPassword {
		t.Fatalf("LoginUser error = %v, want ErrLegacyPassword", err)
	}

	// The legacy retry succeeds and upgrades the account.
	if _, err := LoginUser(LoginRequest{Email: email, Password: plaintext, Legacy: true}); err != nil {
		t.Fatalf("legacy LoginUser: %v", err)
	}

	// Subsequent digest logins work normally.
	if _, err := LoginUser(LoginRequest{Email: email, Password: sha256Hex(plaintext)}); err != nil {
		t.Fatalf("digest LoginUser after migration: %v", err)
	}
}

func TestLoginUserStaleClient(t *testing.T) {
	setupTestDB(t)
	const email = "stale@example.com"
	const plaintext = "stale-pass"
	insertLegacyAccount(t, email, plaintext)

	// A plaintext password without Legacy=true can only come from a cached
	// pre-migration frontend, so the backend must ask for a hard refresh.
	if _, err := LoginUser(LoginRequest{Email: email, Password: plaintext}); err != ErrStaleClient {
		t.Fatalf("LoginUser error = %v, want ErrStaleClient", err)
	}

	// The same applies to an already-migrated account.
	insertLegacyAccount(t, "stale2@example.com", "other-pass")
	if _, err := LoginUser(LoginRequest{Email: "stale2@example.com", Password: "other-pass", Legacy: true}); err != nil {
		t.Fatalf("legacy LoginUser: %v", err)
	}
	if _, err := LoginUser(LoginRequest{Email: "stale2@example.com", Password: "other-pass"}); err != ErrStaleClient {
		t.Fatalf("LoginUser error = %v, want ErrStaleClient for migrated account", err)
	}
}

func TestIsSHA256Hex(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"valid digest", sha256Hex("password"), true},
		{"uppercase hex", "ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789", false},
		{"too short", "abc123", false},
		{"too long", sha256Hex("password") + "0", false},
		{"non-hex chars", "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz", false},
		{"plaintext", "hunter2", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isSHA256Hex(tt.in); got != tt.want {
				t.Fatalf("isSHA256Hex(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
