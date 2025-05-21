package accounts

type RegisterRequest struct {
	DisplayName    string `json:"display_name" binding:"required"`
	Email          string `json:"email" binding:"required,email"`
	HashedPassword string `json:"hashed_password" binding:"required"`
}

type RegisterResponse struct {
	Token string `json:"token"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=3,max=64"`
}

type UserInfo struct {
	Email    string
	Password string
}
