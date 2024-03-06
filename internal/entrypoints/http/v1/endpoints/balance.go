package endpoints

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/dlomanov/go-diploma-tpl/internal/entity"
	"github.com/dlomanov/go-diploma-tpl/internal/infra/deps"
	"github.com/dlomanov/go-diploma-tpl/internal/usecase"
	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
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

// @Router		/api/user/balance [get]
// @Tags		balance
// @Produce	json
// @Param		token	header		string						true	"token with Bearer schema"
// @Success	200		{object}	endpoints.balanceResponse	"ok"
// @Failure	401		{string}	string						"invalid token"
// @Failure	500		{string}	string						"internal server error"
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

// @Router		/api/user/balance/withdraw [post]
// @Tags		balance
// @Accept		json
//
// @Param		request	body		endpoints.withdrawRequest	true	"order number and withdraw sum"
// @Param		token	header		string						true	"token with Bearer schema"
//
// @Success	200		{string}	string						"ok"
// @Failure	401		{string}	string						"invalid creds"
// @Failure	402		{string}	string						"not enough founds"
// @Failure	415		{string}	string						"unsupported content type"
// @Failure	422		{string}	string						"invalid order number"
// @Failure	500		{string}	string						"internal server error"
func (e *balanceEndpoints) withdraw(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		e.logger.Error(NoUserID, zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if v, ok := getContentType(r, ContentTypeJSON); !ok {
		e.logger.Debug(UnsupportedContentType, zap.String("content_type", v))
		w.WriteHeader(http.StatusUnsupportedMediaType)
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
	case errors.Is(err, usecase.ErrOrderExists):
		w.WriteHeader(http.StatusConflict)
	case errors.Is(err, entity.ErrBalanceNotEnoughFounds):
		w.WriteHeader(http.StatusPaymentRequired)
	case err != nil:
		e.logger.Error(InternalError, zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
	default:
		w.WriteHeader(http.StatusOK)
	}
}

// @Router		/api/user/withdrawals [get]
// @Tags		withdrawals
//
// @Param		token	header		string						true	"token with Bearer schema"
//
// @Success	200		{array}		endpoints.withdrawResponse	"ok"
// @Success	204		{string}	string						"no result"
// @Failure	401		{string}	string						"invalid creds"
// @Failure	415		{string}	string						"unsupported content type"
// @Failure	500		{string}	string						"internal server error"
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
