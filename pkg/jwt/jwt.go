package jwt

func GenerateAccessToken(userID string) (string, error) {
	return "dummyAccessTokenFor_" + userID, nil
}

func GenerateRefreshToken(userID string) (string, error) {
	return "dummyRefreshTokenFor_" + userID, nil
}
