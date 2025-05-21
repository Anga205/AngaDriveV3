package accounts

type RegisterRequest struct {
	DisplayName    string `json:"display_name" binding:"required"`
	Email          string `json:"email" binding:"required,email"`
	HashedPassword string `json:"hashed_password" binding:"required"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=3,max=64"`
}

type ChangePasswordRequest struct {
	Email             string `json:"email" binding:"required,email"`
	OldPassword       string `json:"old_password" binding:"required,min=3,max=64"`
	NewPasswordHashed string `json:"new_password_hashed" binding:"required,min=3,max=64"`
}

type ChangeEmailRequest struct {
	OldEmail string `json:"old_email" binding:"required,email"`
	NewEmail string `json:"new_email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=3,max=64"`
}

type ChangeDisplayNameRequest struct {
	DisplayName string `json:"display_name" binding:"required"`
	Email       string `json:"email" binding:"required,email"`
	Password    string `json:"password" binding:"required,min=3,max=64"`
}
