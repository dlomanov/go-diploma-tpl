package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dlomanov/go-diploma-tpl/internal/entity"
	"github.com/dlomanov/go-diploma-tpl/internal/infra/services/pass"
	"github.com/dlomanov/go-diploma-tpl/internal/infra/services/token"
	"github.com/dlomanov/go-diploma-tpl/internal/usecase"
	"github.com/dlomanov/go-diploma-tpl/internal/usecase/mocks"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
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

	tokener := token.NewJWT([]byte("test"), time.Minute)
	balanceRepo := mocks.NewMockBalanceRepo()
	uc := usecase.NewAuthUseCase(
		zap.NewNop(),
		mocks.NewMockUserRepo(),
		balanceRepo,
		pass.NewHasher(0),
		tokener,
		mocks.NewMockTrm(),
	)
	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				gotToken entity.Token
				err      error
			)

			switch tt.args.action {
			case ActionRegister:
				gotToken, err = uc.Register(ctx, tt.args.creds)
			case ActionLogin:
				gotToken, err = uc.Login(ctx, tt.args.creds)
			default:
				t.Fatalf("unknown action type: %s", tt.args.action)
			}

			if errors.Is(err, tt.want.err) {
				return
			}

			require.NoErrorf(t, err, "%s: unexpected error occured: '%v'", tt.args.action, err)
			require.NotEmptyf(t, gotToken, "%s: token should not be empty", tt.args.action)

			userID, err := tokener.GetUserID(gotToken)
			require.NoErrorf(t, err, "%s: error '%v' occured while extracting userID from token", tt.args.action, err)
			require.NotEmptyf(t, uuid.UUID(userID), "%s: userID should not be empty", tt.args.action)

			balance, err := balanceRepo.Get(ctx, userID)
			require.NoErrorf(t, err, "%s: error '%v' occured while getting balance", tt.args.action, err)
			require.True(t, balance.Current.Equal(decimal.Zero), "%s: current balance should be zero after creation", tt.args.action)
			require.True(t, balance.Withdrawn.Equal(decimal.Zero), "%s: withdrawn balance should be zero after creation", tt.args.action)
		})
	}
}
