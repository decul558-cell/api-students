package model

import "time"

type RegisterRequest struct {
	Username string `json:"username" validate:"required,min=3,max=30,alphanum"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,strongpassword"`
	// Perhatikan: TIDAK ADA field Role di sini. Bila ada, siapa pun
	// dapat mendaftar sebagai admin (kerentanan mass assignment).
}

// LoginRequest sengaja TIDAK memakai tag strongpassword. Aturan kekuatan
// password hanya berlaku saat MEMBUAT password baru (register), bukan
// saat memverifikasi password yang sudah ada — user lama mungkin punya
// password yang dibuat sebelum aturan kekuatan ini ada.
type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

// RefreshToken adalah row pada tabel refresh_tokens.
type RefreshToken struct {
	ID        int64
	UserID    int
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

type AuthUser struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}
