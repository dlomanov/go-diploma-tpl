package middlewares

import (
	"github.com/dlomanov/go-diploma-tpl/internal/deps"
	"github.com/dlomanov/go-diploma-tpl/internal/entity"
	"github.com/google/uuid"
	"net/http"
	"strings"
)

const (
	UserIDHeader = "X-User-ID"
	authHeader   = "Authorization"
	authSchema   = "Bearer "
)

func Auth(c *deps.Container) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get(authHeader)
			if !strings.HasPrefix(h, authSchema) {
				c.Logger.Debug("request without authorization header")
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			token := h[len(authSchema):]
			userID, err := c.Tokener.GetUserID(entity.Token(token))
			if err != nil {
				c.Logger.Debug("request with invalid token")
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			r.Header.Set(UserIDHeader, uuid.UUID(userID).String())

			next.ServeHTTP(w, r)
		})
	}
}