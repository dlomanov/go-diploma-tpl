package endpoints

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/dlomanov/go-diploma-tpl/internal/entity"
	"github.com/dlomanov/go-diploma-tpl/internal/infra/deps"
	"github.com/dlomanov/go-diploma-tpl/internal/usecase"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type (
	authEndpoints struct {
		logger      *zap.Logger
		authUseCase *usecase.AuthUseCase
	}
	loginRequest struct {
		Login string `json:"login"`
		Pass  string `json:"password"`
	}
	loginErrorResponse struct {
		Errors []string `json:"validation_errors"`
	}
)

func UseAuthEndpoints(router chi.Router, c *deps.Container) {
	e := &authEndpoints{
		logger:      c.Logger,
		authUseCase: c.AuthUseCase,
	}

	router.Post("/api/user/register", e.register)
	router.Post("/api/user/login", e.login)
}

// @Router		/api/user/register [post]
//
// @Tags		auth
// @Accept		json
//
// @Param		request	body		endpoints.loginRequest			true	"user creds"
//
// @Success	200		{string}	string							"ok"
// @Failure	400		{object}	endpoints.loginErrorResponse	"validation failed"
// @Failure	401		{string}	string							"invalid creds"
// @Failure	415		{string}	string							"unsupported content type"
// @Failure	500		{string}	string							"internal server error"
//
// @Header		200		{string}	Authorization					"<schema> <token>"
func (e *authEndpoints) register(w http.ResponseWriter, r *http.Request) {
	if h, ok := getContentType(r, ContentTypeJSON); !ok {
		e.logger.Debug(UnsupportedContentType, zap.String("content_type", h))
		w.WriteHeader(http.StatusUnsupportedMediaType)
		return
	}

	model := new(loginRequest)
	if err := json.NewDecoder(r.Body).Decode(model); err != nil {
		e.logger.Error("failed to decode json request", zap.Error(err))
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

// @Router		/api/user/login [post]
// @Tags		auth
//
// @Param		request	body		endpoints.loginRequest			true	"user creds"
//
// @Success	200		{string}	string							"ok"
// @Failure	400		{object}	endpoints.loginErrorResponse	"validation failed"
// @Failure	401		{string}	string							"invalid creds"
// @Failure	415		{string}	string							"unsupported content type"
// @Failure	500		{string}	string							"internal server error"
//
// @Header		200		{string}	Authorization					"<schema> <token>"
func (e *authEndpoints) login(w http.ResponseWriter, r *http.Request) {
	if h, ok := getContentType(r, ContentTypeJSON); !ok {
		e.logger.Debug(UnsupportedContentType, zap.String("content_type", h))
		w.WriteHeader(http.StatusUnsupportedMediaType)
		return
	}

	model := new(loginRequest)
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

func (req loginRequest) validate() []error {
	var errs = make([]error, 0)
	if req.Login == "" {
		errs = append(errs, fmt.Errorf("login must be specified"))
	}
	if req.Pass == "" {
		errs = append(errs, fmt.Errorf("password must be specified"))
	}

	return errs
}

func (req loginRequest) toEntity() entity.Creds {
	return entity.Creds{
		Login: entity.Login(req.Login),
		Pass:  entity.Pass(req.Pass),
	}
}

func newResponse(errs []error) loginErrorResponse {
	errstrs := make([]string, len(errs))
	for i, err := range errs {
		errstrs[i] = err.Error()
	}

	return loginErrorResponse{
		Errors: errstrs,
	}
}

func (e *authEndpoints) writeInvalid(w http.ResponseWriter, errs []error) {
	e.logger.Debug("validation failed", zap.Errors("validation_errors", errs))
	w.Header().Set(ContentTypeHeader, ContentTypeJSON)
	w.WriteHeader(http.StatusBadRequest)
	resp := newResponse(errs)
	err := json.NewEncoder(w).Encode(resp)
	if err != nil {
		e.logger.Error("failed to marshal response", zap.Error(err))
	}
}

func (e *authEndpoints) writeToken(w http.ResponseWriter, token entity.Token) {
	w.Header().Set(AuthorizationHeader, fmt.Sprintf("Bearer %s", token))
	w.WriteHeader(http.StatusOK)
}
