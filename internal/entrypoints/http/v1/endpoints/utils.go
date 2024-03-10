package endpoints

import (
	"net/http"
	"strings"

	"github.com/dlomanov/go-diploma-tpl/internal/entity"
	"github.com/dlomanov/go-diploma-tpl/internal/entity/apperrors"
	"github.com/dlomanov/go-diploma-tpl/internal/entrypoints/http/middlewares"
	"github.com/google/uuid"
)

const (
	AuthorizationHeader = "Authorization"
	ContentTypeHeader   = "Content-Type"
	ContentTypeJSON     = "application/json"
	ContentTypeText     = "text/plain"

	NoUserID               = "UserID not defined"
	InternalError          = "unexpected internal error"
	UnsupportedContentType = "unsupported content type"
)

func getContentType(r *http.Request, contentType string) (string, bool) {
	if h := r.Header.Get(ContentTypeHeader); strings.HasPrefix(h, contentType) {
		return h, true
	}
	return "", false
}

func getUserID(r *http.Request) (entity.UserID, error) {
	h := r.Header.Get(middlewares.UserIDHeader)
	if h == "" {
		return entity.UserID{}, apperrors.NewInternal(NoUserID)
	}

	userID, err := uuid.Parse(h)
	if err != nil {
		return entity.UserID{}, err
	}

	return entity.UserID(userID), nil
}
