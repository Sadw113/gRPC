package service

import (
	sso "gRPC/gRPC/genGo"
	"gRPC/internal/repo/mocks"
	"testing"
)

func TestRegister(t *testing.T) {
	type testTable struct {
		repo mocks.Repository
	}

	tests := []struct {
		name        string
		args        testTable
		expectedRes sso.RegisterResponse
	}{
		{
			name:        "ok",
			expectedRes: sso.RegisterResponse{Message: "Creating a new user with id 1 was successfull"},
		},
	}
	for _, cases := range tests {
		t.Run(cases.name, func(t *testing.T) {

		})
	}
}
