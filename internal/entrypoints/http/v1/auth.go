package v1

import (
	"encoding/json"
	"fmt"
	"github.com/dlomanov/go-diploma-tpl/internal/deps"
	"github.com/dlomanov/go-diploma-tpl/internal/entity"
	"github.com/dlomanov/go-diploma-tpl/internal/usecase"
	"github.com/go-chi/chi/v5"
	"github.com/go-errors/errors"
	"go.uber.org/zap"
	"net/http"
	"strings"
)

type endpoints struct {
	logger      *zap.Logger
	authUseCase *usecase.AuthUseCase
}

func UseAuthEndpoints(router chi.Router, c *deps.Container) {
	r := &endpoints{
		logger:      c.Logger,
		authUseCase: c.AuthUseCase,
	}

	router.Post("/api/user/register", r.Register)
}

type registerRequest struct {
	Login string `json:"login"`
	Pass  string `json:"password"`
}

func (request registerRequest) Validate() []error {
	var errs = make([]error, 0)
	if request.Login == "" {
		errs = append(errs, fmt.Errorf("login must be specified"))
	}
	if request.Pass == "" {
		errs = append(errs, fmt.Errorf("password must be specified"))
	}

	return errs
}

type registerResponse struct {
	Errors []string `json:"validation_errors"`
}

func newRegisterResponse(errs []error) registerResponse {
	errstrs := make([]string, len(errs))
	for i, err := range errs {
		errstrs[i] = err.Error()
	}

	return registerResponse{
		Errors: errstrs,
	}
}

func (ar *endpoints) Register(w http.ResponseWriter, r *http.Request) {
	if h := r.Header.Get("Content-Type"); !strings.HasPrefix(h, "application/json") {
		ar.logger.Debug("unsupported content type", zap.String("content_type", h))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	model := new(registerRequest)
	if err := json.NewDecoder(r.Body).Decode(model); err != nil {
		ar.logger.Error("failed to unmarshal json request", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if errs := model.Validate(); len(errs) != 0 {
		ar.logger.Debug("validation failed", zap.Errors("validation_errors", errs))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		response := newRegisterResponse(errs)
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			ar.logger.Error("failed to marshal response", zap.Error(err))
		}
	}

	token, err := ar.authUseCase.Register(r.Context(),
		entity.Creds{
			Login: entity.Login(model.Login),
			Pass:  entity.Pass(model.Pass),
		})
	if errors.Is(err, usecase.ErrAuthUserExists) {
		ar.logger.Debug("user already exists",
			zap.String("login", model.Login),
			zap.Error(err))
		w.WriteHeader(http.StatusConflict)
		return
	}
	if err != nil {
		ar.logger.Debug("failed to register user",
			zap.String("login", model.Login),
			zap.Error(err))
		w.WriteHeader(http.StatusConflict)
	}

	w.Header().Set("Authorization", "Bearer "+string(token))
	w.WriteHeader(http.StatusOK)
}
