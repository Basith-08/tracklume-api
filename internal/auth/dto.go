package auth

type RegisterRequest struct {
	Name     string `json:"name" validate:"required,min=1,max=120"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=128"`
}
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}
type UpdateProfileRequest struct {
	Name      *string `json:"name" validate:"omitempty,min=1,max=120"`
	AvatarURL *string `json:"avatar_url" validate:"omitempty,max=500"`
}
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8,max=128"`
}

type UserResponse struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Email     string  `json:"email"`
	AvatarURL *string `json:"avatar_url"`
	CreatedAt string  `json:"created_at,omitempty"`
	UpdatedAt string  `json:"updated_at,omitempty"`
}
type LoginResponse struct {
	AccessToken string       `json:"access_token"`
	TokenType   string       `json:"token_type"`
	ExpiresIn   int64        `json:"expires_in"`
	User        UserResponse `json:"user"`
}

func presentUser(user User) UserResponse {
	return UserResponse{ID: user.ID.String(), Name: user.Name, Email: user.Email, AvatarURL: user.AvatarURL, CreatedAt: user.CreatedAt.UTC().Format("2006-01-02T15:04:05.000Z07:00"), UpdatedAt: user.UpdatedAt.UTC().Format("2006-01-02T15:04:05.000Z07:00")}
}
