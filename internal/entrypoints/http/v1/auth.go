package v1

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dlomanov/go-diploma-tpl/internal/deps"
	"github.com/dlomanov/go-diploma-tpl/internal/entity"
	"github.com/dlomanov/go-diploma-tpl/internal/usecase"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"net/http"
	"strings"
)

const (
	AuthorizationHeader = "Authorization"
	ContentTypeHeader   = "Content-Type"
	ContentTypeJSON     = "application/json"
)

type endpoints struct {
	logger      *zap.Logger
	authUseCase *usecase.AuthUseCase
}

func UseAuthEndpoints(router chi.Router, c *deps.Container) {
	e := &endpoints{
		logger:      c.Logger,
		authUseCase: c.AuthUseCase,
	}

	router.Post("/api/user/register", e.Register)
	router.Post("/api/user/login", e.Login)
}

func (e *endpoints) Register(w http.ResponseWriter, r *http.Request) {
	if h, ok := get(ContentTypeJSON, r); !ok {
		e.logger.Debug("unsupported content type", zap.String("content_type", h))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	model := new(request)
	if err := json.NewDecoder(r.Body).Decode(model); err != nil {
		e.logger.Error("failed to unmarshal json request", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if errs := model.validate(); len(errs) != 0 {
		e.writeInvalid(w, errs)
		return
	}

	token, err := e.authUseCase.Register(r.Context(), model.toEntity())
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrAuthUserExists):
			e.logger.Debug("user already exists", zap.String("login", model.Login), zap.Error(err))
			w.WriteHeader(http.StatusConflict)
		default:
			e.logger.Debug("failed to login user", zap.String("login", model.Login), zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	e.writeToken(w, token)
}

func (e *endpoints) Login(w http.ResponseWriter, r *http.Request) {
	if h, ok := get(ContentTypeJSON, r); !ok {
		e.logger.Debug("unsupported content type", zap.String("content_type", h))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	model := new(request)
	if err := json.NewDecoder(r.Body).Decode(model); err != nil {
		e.logger.Error("failed to unmarshal json request", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if errs := model.validate(); len(errs) != 0 {
		e.writeInvalid(w, errs)
		return
	}

	token, err := e.authUseCase.Login(r.Context(), model.toEntity())
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrAuthUserNotFound):
			e.logger.Debug("user not found", zap.String("login", model.Login), zap.Error(err))
			w.WriteHeader(http.StatusUnauthorized)
		case errors.Is(err, usecase.ErrAuthUserInvalidCreds):
			e.logger.Debug("invalid user password", zap.String("login", model.Login), zap.Error(err))
			w.WriteHeader(http.StatusUnauthorized)
		default:
			e.logger.Debug("failed to login user", zap.String("login", model.Login), zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	e.writeToken(w, token)
}

type request struct {
	Login string `json:"login"`
	Pass  string `json:"password"`
}

type response struct {
	Errors []string `json:"validation_errors"`
}

func (req request) validate() []error {
	var errs = make([]error, 0)
	if req.Login == "" {
		errs = append(errs, fmt.Errorf("login must be specified"))
	}
	if req.Pass == "" {
		errs = append(errs, fmt.Errorf("password must be specified"))
	}

	return errs
}

func (req request) toEntity() entity.Creds {
	return entity.Creds{
		Login: entity.Login(req.Login),
		Pass:  entity.Pass(req.Pass),
	}
}

func newResponse(errs []error) response {
	errstrs := make([]string, len(errs))
	for i, err := range errs {
		errstrs[i] = err.Error()
	}

	return response{
		Errors: errstrs,
	}
}

func get(contentType string, r *http.Request) (string, bool) {
	if h := r.Header.Get(ContentTypeHeader); strings.HasPrefix(h, contentType) {
		return h, true
	}
	return "", false
}

func (e *endpoints) writeInvalid(w http.ResponseWriter, errs []error) {
	e.logger.Debug("validation failed", zap.Errors("validation_errors", errs))
	w.Header().Set(ContentTypeHeader, ContentTypeJSON)
	w.WriteHeader(http.StatusBadRequest)
	resp := newResponse(errs)
	err := json.NewEncoder(w).Encode(resp)
	if err != nil {
		e.logger.Error("failed to marshal response", zap.Error(err))
	}
}

func (e *endpoints) writeToken(w http.ResponseWriter, token entity.Token) {
	w.Header().Set(AuthorizationHeader, fmt.Sprintf("Bearer %s", token))
	w.WriteHeader(http.StatusOK)
}
