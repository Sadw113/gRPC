package jwt

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func GenerateAccessToken(userID string) (string, error) {
	payload := jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(time.Hour * 72).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)

	t, err := token.SignedString("dummyAccessTokenFor_")
	if err != nil {
		return "", err
	}

	return t, nil
}

func GenerateRefreshToken(userID string) (string, error) {
	payload := jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(time.Hour * 72).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)

	t, err := token.SignedString("dummyRefreshTokenFor_")
	if err != nil {
		return "", err
	}

	return t, nil
}
