package endpoints

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/dlomanov/go-diploma-tpl/internal/entity"
	"github.com/dlomanov/go-diploma-tpl/internal/infra/deps"
	"github.com/dlomanov/go-diploma-tpl/internal/usecase"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type (
	orderEndpoints struct {
		logger       *zap.Logger
		orderUseCase *usecase.OrderUseCase
	}
	orderResponse struct {
		Number     string `json:"number,omitempty"`
		Status     string `json:"status,omitempty"`
		Accrual    string `json:"accrual,omitempty"`
		UploadedAt string `json:"uploaded_at,omitempty"`
	}
)

func UseOrderEndpoints(router chi.Router, c *deps.Container) {
	e := &orderEndpoints{
		logger:       c.Logger,
		orderUseCase: c.OrderUseCase,
	}

	router.Post("/api/user/orders", e.createOrder)
	router.Get("/api/user/orders", e.getOrders)
}

// @Router		/api/user/orders [post]
// @Tags		orders
// @Accept		plain
//
// @Param		request	body		string	true	"order number"
// @Param		token	header		string	true	"token with Bearer schema"
//
// @Success	200		{string}	string	"order processing"
// @Success	202		{string}	string	"order accepted"
// @Failure	401		{string}	string	"invalid creds"
// @Failure	415		{string}	string	"unsupported content type"
// @Failure	500		{string}	string	"internal server error"
func (e *orderEndpoints) createOrder(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		e.logger.Error(NoUserID, zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if v, ok := getContentType(r, ContentTypeText); !ok {
		e.logger.Debug(UnsupportedContentType, zap.String("content_type", v))
		w.WriteHeader(http.StatusUnsupportedMediaType)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		e.logger.Error("read body failed", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	number := entity.OrderNumber(body)

	err = e.orderUseCase.Create(r.Context(), number, userID)
	switch {
	case errors.Is(err, usecase.ErrOrderExists):
		w.WriteHeader(http.StatusOK)
	case errors.Is(err, usecase.ErrOrderSavedByAnotherUser):
		w.WriteHeader(http.StatusConflict)
	case errors.Is(err, usecase.ErrOrderNumberInvalid):
		w.WriteHeader(http.StatusUnprocessableEntity)
	case err != nil:
		e.logger.Error(InternalError, zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
	default:
		w.WriteHeader(http.StatusAccepted)
	}
}

// @Router		/api/user/orders [get]
// @Tags		orders
// @Produce	json
//
// @Param		token	header		string					true	"token with Bearer schema"
//
// @Success	200		{array}		endpoints.orderResponse	"orders"
// @Success	204		{string}	string					"no result"
// @Failure	401		{string}	string					"invalid creds"
// @Failure	415		{string}	string					"unsupported content type"
// @Failure	500		{string}	string					"internal server error"
func (e *orderEndpoints) getOrders(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		e.logger.Error(NoUserID, zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	orders, err := e.orderUseCase.GetAll(r.Context(), userID)
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

func (*orderEndpoints) toResponse(orders []entity.Order) []orderResponse {
	res := make([]orderResponse, len(orders))
	for i, v := range orders {
		or := orderResponse{
			Number:     string(v.Number),
			Status:     string(v.Status),
			UploadedAt: v.CreatedAt.Format(time.RFC3339),
		}
		if v.Status == entity.OrderStatusProcessed {
			or.Accrual = v.Amount.String()
		}
		res[i] = or
	}
	return res
}
