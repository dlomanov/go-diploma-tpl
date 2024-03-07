package entity

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewUser(t *testing.T) {
	creds := HashCreds{
		Login:    "testUser",
		PassHash: "hashedPassword",
	}

	user, err := NewUser(creds)
	require.NoError(t, err, "error creating new user")
	require.NotNil(t, user, "user should not be nil")
	require.Equal(t, creds, user.HashCreds, "hash creds mismatch")
	require.NotZero(t, user.ID, "user ID should not be zero")
}

func TestTokenValid(t *testing.T) {
	tests := []struct {
		name  string
		token Token
		want  bool
	}{
		{
			name:  "valid token",
			token: Token("validToken"),
			want:  true,
		},
		{
			name:  "invalid token",
			token: Token(""),
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
		creds Creds
		want  bool
	}{
		{
			name: "valid creds",
			creds: Creds{
				Login: "testUser",
				Pass:  "password",
			},
			want: true,
		},
		{
			name: "invalid creds",
			creds: Creds{
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
