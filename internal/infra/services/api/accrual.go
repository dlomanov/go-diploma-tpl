package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dlomanov/go-diploma-tpl/internal/entity"
	"github.com/dlomanov/go-diploma-tpl/internal/entity/apperrors"
	"github.com/dlomanov/go-diploma-tpl/internal/usecase"
	"github.com/go-resty/resty/v2"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

const (
	defaultRetryAfter = 1 * time.Second
	retryAfterHeader  = "Retry-After"

	accrualStatusRegistered accrualStatus = "REGISTERED"
	accrualStatusInvalid    accrualStatus = "INVALID"
	accrualStatusProcessing accrualStatus = "PROCESSING"
	accrualStatusProcessed  accrualStatus = "PROCESSED"
)

var (
	_ usecase.OrderAccrualAPI = (*AccrualAPI)(nil)

	empty = entity.OrderAccrual{}

	ErrUnexpectedStatusCode    = apperrors.NewInternal("unexpected status code")
	ErrUnexpectedResponse      = apperrors.NewInternal("unexpected response")
	ErrUnexpectedAccrualStatus = apperrors.NewInternal("unexpected accrual status")
	ErrInternalServerError     = apperrors.NewInternal("internal server error")
	ErrOrderNotFound           = apperrors.NewNotFound("order not found")
)

type (
	AccrualAPI struct {
		client *resty.Client
		logger *zap.Logger
	}
	result struct {
		Order   entity.OrderNumber `json:"order"`
		Status  accrualStatus      `json:"status"`
		Accrual decimal.Decimal    `json:"accrual"`
	}
	accrualStatus string
)

func NewAccrualAPI(
	logger *zap.Logger,
	addr string,
) *AccrualAPI {
	if !strings.HasPrefix(addr, "http") {
		addr = "http://" + addr
	}
	client := resty.New()
	client.SetBaseURL(addr)

	return &AccrualAPI{
		client: client,
		logger: logger,
	}

}

func (a *AccrualAPI) Get(
	ctx context.Context,
	number entity.OrderNumber,
) (entity.OrderAccrual, error) {
	resp, err := a.client.R().
		SetContext(ctx).
		SetResult(result{}).
		SetPathParam("number", string(number)).
		Get("api/orders/{number}")

	if err != nil {
		return empty, err
	}

	switch resp.StatusCode() {
	case http.StatusOK:
		return a.toEntity(resp)
	case http.StatusNoContent:
		return empty, fmt.Errorf("%w, number: '%s'", ErrOrderNotFound, number)
	case http.StatusTooManyRequests:
		return empty, a.toTransient(resp)
	case http.StatusInternalServerError:
		return empty, ErrInternalServerError
	default:
		err = fmt.Errorf("%w: %d", ErrUnexpectedStatusCode, resp.StatusCode())
		return empty, err
	}
}

func (a *AccrualAPI) toEntity(
	resp *resty.Response,
) (entity.OrderAccrual, error) {
	r := resp.Result().(*result)
	if r == nil {
		a.logger.Error("accrual response empty")
		return empty, ErrUnexpectedResponse
	}

	var s entity.OrderStatus
	switch r.Status {
	case accrualStatusRegistered:
		s = entity.OrderStatusProcessing
	case accrualStatusInvalid:
		s = entity.OrderStatusInvalid
	case accrualStatusProcessing:
		s = entity.OrderStatusProcessing
	case accrualStatusProcessed:
		s = entity.OrderStatusProcessed
	default:
		a.logger.Error("unexpected accrual status", zap.String("status", string(r.Status)))
		return empty, ErrUnexpectedAccrualStatus
	}

	return entity.OrderAccrual{
		Amount: entity.Amount(r.Accrual),
		Status: s,
	}, nil
}

func (a *AccrualAPI) toTransient(resp *resty.Response) error {
	after := defaultRetryAfter
	if v := resp.Header().Get(retryAfterHeader); v != "" {
		afterSec, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			a.logger.Error("failed to parse Retry-After header", zap.Error(err))
			return err
		}
		after = time.Duration(afterSec * int64(time.Second))
	}
	return apperrors.NewTransient("too many requests", after)
}
