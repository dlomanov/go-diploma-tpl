package usecase_test

import (
	"context"
	"github.com/dlomanov/go-diploma-tpl/internal/entity"
	"github.com/dlomanov/go-diploma-tpl/internal/usecase"
	"github.com/dlomanov/go-diploma-tpl/internal/usecase/pass"
	"github.com/dlomanov/go-diploma-tpl/internal/usecase/token"
	"github.com/go-errors/errors"
	"testing"
	"time"
)

const (
	ActionRegister = "register"
	ActionLogin    = "login"
)

func TestAuthUseCase(t *testing.T) {
	type args struct {
		action string
		creds  entity.Creds
	}
	type want struct {
		err error
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "fail: empty login",
			args: args{
				action: ActionRegister,
				creds: entity.Creds{
					Login: "",
					Pass:  "1",
				},
			},
			want: want{err: usecase.ErrAuthUserInvalidCreds},
		},
		{
			name: "fail: empty password",
			args: args{
				action: ActionRegister,
				creds: entity.Creds{
					Login: "admin",
					Pass:  "",
				},
			},
			want: want{err: usecase.ErrAuthUserInvalidCreds},
		},
		{
			name: "fail empty creds",
			args: args{
				action: ActionRegister,
				creds: entity.Creds{
					Login: "",
					Pass:  "",
				},
			},
			want: want{err: usecase.ErrAuthUserInvalidCreds},
		},
		{
			name: "success: user registered",
			args: args{
				action: ActionRegister,
				creds: entity.Creds{
					Login: "admin",
					Pass:  "1",
				},
			},
			want: want{err: nil},
		},
		{
			name: "failed: user already registered",
			args: args{
				action: ActionRegister,
				creds: entity.Creds{
					Login: "admin",
					Pass:  "1",
				},
			},
			want: want{err: usecase.ErrAuthUserExists},
		},
		{
			name: "success: user registered",
			args: args{
				action: ActionRegister,
				creds: entity.Creds{
					Login: "admin2",
					Pass:  "1",
				},
			},
			want: want{err: nil},
		},
		{
			name: "fail: empty login",
			args: args{
				action: ActionLogin,
				creds: entity.Creds{
					Login: "admin",
					Pass:  "1",
				},
			},
			want: want{err: usecase.ErrAuthUserInvalidCreds},
		},
		{
			name: "fail: empty password",
			args: args{
				action: ActionLogin,
				creds: entity.Creds{
					Login: "admin",
					Pass:  "",
				},
			},
			want: want{err: usecase.ErrAuthUserInvalidCreds},
		},
		{
			name: "fail: empty creds",
			args: args{
				action: ActionLogin,
				creds: entity.Creds{
					Login: "admin",
					Pass:  "",
				},
			},
			want: want{err: usecase.ErrAuthUserInvalidCreds},
		},
		{
			name: "success: user logged in",
			args: args{
				action: ActionLogin,
				creds: entity.Creds{
					Login: "admin",
					Pass:  "1",
				},
			},
			want: want{err: nil},
		},
	}

	uc := usecase.NewAuth(
		newMockUserRepo(),
		pass.NewHasher(0),
		token.NewJWT([]byte("test"), time.Minute))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				gotToken entity.Token
				err      error
			)

			switch tt.args.action {
			case ActionRegister:
				gotToken, err = uc.Register(context.Background(), tt.args.creds)
			case ActionLogin:
				gotToken, err = uc.Login(context.Background(), tt.args.creds)
			default:
				t.Fatalf("unknown action type: %s", tt.args.action)
			}

			if errors.Is(err, tt.want.err) {
				return
			} else if err != nil {
				t.Fatalf("%s unexpected error: %v", tt.args.action, err)
				return
			}

			if gotToken == "" {
				t.Fatalf("%s empty token", tt.args.action)
			}
		})
	}
}
