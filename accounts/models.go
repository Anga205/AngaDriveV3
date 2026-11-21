package accounts

type RegisterRequest struct {
	DisplayName    string `json:"display_name" binding:"required"`
	Email          string `json:"email" binding:"required,email"`
	HashedPassword string `json:"hashed_password" binding:"required"`
}

type LoginRequest struct {
	Email string `json:"email" binding:"required,email"`
	// Password is the hex SHA-256 digest of the user's password for migrated
	// accounts, or the plaintext password when Legacy is true.
	Password string `json:"password" binding:"required,min=3,max=128"`
	// Legacy asks the backend to verify Password as plaintext against a
	// pre-migration (bcrypt-of-plaintext) account and upgrade it in place.
	Legacy bool `json:"legacy"`
}

type DeleteUserRequest LoginRequest

type ChangePasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
	// OldPassword is the hex SHA-256 digest of the current password.
	OldPassword string `json:"old_password" binding:"required,min=3,max=128"`
	// NewPasswordHashed is bcrypt(hex SHA-256 digest of the new password).
	NewPasswordHashed string `json:"new_password_hashed" binding:"required,min=3,max=128"`
}

type ChangeEmailRequest struct {
	OldEmail string `json:"old_email" binding:"required,email"`
	NewEmail string `json:"new_email" binding:"required,email"`
	// Password is the hex SHA-256 digest of the current password.
	Password string `json:"password" binding:"required,min=3,max=128"`
}

type ChangeDisplayNameRequest struct {
	DisplayName string `json:"display_name" binding:"required"`
	Email       string `json:"email" binding:"required,email"`
	// Password is the hex SHA-256 digest of the current password.
	Password string `json:"password" binding:"required,min=3,max=128"`
}

// MigratePasswordRequest upgrades a legacy account (bcrypt of the plaintext
// password) to the current scheme (bcrypt of the hex SHA-256 digest).
type MigratePasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
	// OldPassword is the plaintext password cached by the legacy frontend.
	OldPassword string `json:"old_password" binding:"required,min=3,max=64"`
	// NewPasswordHashed is bcrypt(hex SHA-256 digest of OldPassword).
	NewPasswordHashed string `json:"new_password_hashed" binding:"required,min=3,max=128"`
}
