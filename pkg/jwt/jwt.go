package jwt

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func GenerateAccessToken(userID string, accesSecret string) (string, error) {
	payload := jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(time.Minute * 30).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)

	t, err := token.SignedString([]byte(accesSecret))
	if err != nil {
		return "", err
	}

	return t, nil
}

func GenerateRefreshToken(userID string, refreshSecret string) (string, error) {
	payload := jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(time.Hour * 72).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)

	t, err := token.SignedString([]byte(refreshSecret))
	if err != nil {
		return "", err
	}

	return t, nil
}
