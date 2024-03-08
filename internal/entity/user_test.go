package entity_test

import (
	"testing"

	"github.com/dlomanov/go-diploma-tpl/internal/entity"
	"github.com/stretchr/testify/require"
)

func TestNewUser(t *testing.T) {
	creds := entity.HashCreds{
		Login:    "testUser",
		PassHash: "hashedPassword",
	}

	user, err := entity.NewUser(creds)
	require.NoError(t, err, "error creating new user")
	require.NotNil(t, user, "user should not be nil")
	require.Equal(t, creds, user.HashCreds, "hash creds mismatch")
	require.NotZero(t, user.ID, "user ID should not be zero")
}

func TestTokenValid(t *testing.T) {
	tests := []struct {
		name  string
		token entity.Token
		want  bool
	}{
		{
			name:  "valid token",
			token: entity.Token("validToken"),
			want:  true,
		},
		{
			name:  "invalid token",
			token: entity.Token(""),
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.token.Valid(), "token validity mismatch")
		})
	}
}

func TestCredsValid(t *testing.T) {
	tests := []struct {
		name  string
		creds entity.Creds
		want  bool
	}{
		{
			name: "valid creds",
			creds: entity.Creds{
				Login: "testUser",
				Pass:  "password",
			},
			want: true,
		},
		{
			name: "invalid creds",
			creds: entity.Creds{
				Login: "",
				Pass:  "password",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.creds.Valid(), "creds validity mismatch")
		})
	}
}
