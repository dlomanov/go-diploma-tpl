package entity

import "github.com/google/uuid"

type User struct {
	ID UserID
	HashCreds
}

type Creds struct {
	Login Login
	Pass  Pass
}

type HashCreds struct {
	Login    Login
	PassHash PassHash
}

type UserToken struct {
	ID    UserID
	Token Token
}

type Login string
type Pass string
type PassHash string
type Token string
type UserID uuid.UUID
