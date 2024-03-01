package endpoints

import (
	"encoding/json"
	"errors"
	"github.com/dlomanov/go-diploma-tpl/internal/deps"
	"github.com/dlomanov/go-diploma-tpl/internal/entity"
	"github.com/dlomanov/go-diploma-tpl/internal/usecase"
	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"net/http"
	"time"
)

type (
	balanceEndpoints struct {
		logger         *zap.Logger
		balanceUseCase *usecase.BalanceUseCase
	}
	balanceResponse struct {
		Current   string `json:"current,omitempty"`
		Withdrawn string `json:"withdrawn,omitempty"`
	}
	withdrawRequest struct {
		Order string          `json:"order"`
		Sum   decimal.Decimal `json:"sum"`
	}
	withdrawResponse struct {
		Order       string `json:"order"`
		Sum         string `json:"sum"`
		ProcessedAt string `json:"processed_at"`
	}
)

func UseBalanceEndpoints(router chi.Router, c *deps.Container) {
	e := &balanceEndpoints{
		logger:         c.Logger,
		balanceUseCase: c.BalanceUseCase,
	}
	router.Get("/api/user/balance", e.getBalance)
	router.Post("/api/user/balance/withdraw", e.withdraw)
	router.Get("/api/user/withdrawals", e.getWithdrawals)
}

func (e *balanceEndpoints) getBalance(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		e.logger.Error(NoUserID, zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	balance, err := e.balanceUseCase.GetBalance(r.Context(), userID)
	switch {
	case err != nil:
		e.logger.Error(InternalError, zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
	default:
		w.Header().Set(ContentTypeHeader, ContentTypeJSON)
		if err = json.NewEncoder(w).Encode(balanceResponse{
			Current:   balance.Current.String(),
			Withdrawn: balance.Withdrawn.String(),
		}); err != nil {
			e.logger.Error("failed to writer JSON response", zap.Error(err))
		}
	}
}

func (e *balanceEndpoints) withdraw(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		e.logger.Error(NoUserID, zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if v, ok := getContentType(r, ContentTypeText); !ok {
		e.logger.Debug("unsupported content type", zap.String("content_type", v))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	model := new(withdrawRequest)
	if err = json.NewDecoder(r.Body).Decode(model); err != nil {
		e.logger.Error("failed to decode json request", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = e.balanceUseCase.Withdraw(
		r.Context(),
		entity.OrderNumber(model.Order),
		entity.Amount(model.Sum),
		userID)
	switch {
	case errors.Is(err, usecase.ErrOrderNumberInvalid):
		w.WriteHeader(http.StatusUnprocessableEntity)
	case errors.Is(err, entity.ErrBalanceNotEnoughFounds):
		w.WriteHeader(http.StatusPaymentRequired)
	case err != nil:
		e.logger.Error(InternalError, zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
	default:
		w.WriteHeader(http.StatusOK)
	}
}

func (e *balanceEndpoints) getWithdrawals(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		e.logger.Error(NoUserID, zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	orders, err := e.balanceUseCase.GetWithdrawals(r.Context(), userID)
	switch {
	case err != nil:
		e.logger.Error(InternalError, zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
	case len(orders) == 0:
		w.WriteHeader(http.StatusNoContent)
	default:
		resp := e.toResponse(orders)
		w.Header().Set(ContentTypeHeader, ContentTypeJSON)
		enc := json.NewEncoder(w)
		for _, v := range resp {
			err = enc.Encode(v)
			if err != nil {
				e.logger.Error("failed to write JSON response", zap.Error(err))
				break
			}
		}
	}
}

func (*balanceEndpoints) toResponse(orders []entity.Order) []withdrawResponse {
	res := make([]withdrawResponse, len(orders))
	for i, v := range orders {
		res[i] = withdrawResponse{
			Order:       string(v.Number),
			Sum:         v.Amount.String(),
			ProcessedAt: v.UpdatedAt.Format(time.RFC3339),
		}
	}
	return res
}
