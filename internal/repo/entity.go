package repo

import "time"

type User struct {
	ID             int64
	Username       string
	HashedPassword string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type User_Tokens struct {
	User_ID      int64
	AccessToken  string
	RefreshToken string
}

type NewRefreshTokenParams struct {
	UserID int64
	Token  string
}

type UpdatePasswordData struct {
	Username    string
	NewPassword string
}
