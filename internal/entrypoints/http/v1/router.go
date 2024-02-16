package v1

import (
	"github.com/dlomanov/go-diploma-tpl/internal/deps"
	"github.com/dlomanov/go-diploma-tpl/internal/entrypoints/http/middlewares"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"net/http"
)

func NewRouter(r chi.Router, c *deps.Container) {
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	UseAuthEndpoints(r, c)

	r.Group(func(r chi.Router) {
		r.Use(middlewares.Auth(c))
		r.Get("/api/ping", func(w http.ResponseWriter, r *http.Request) {
			userID := r.Header.Get(middlewares.UserIDHeader)
			if userID == "" {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte("NO USER_ID"))
				return
			}

			_, _ = w.Write([]byte("USER_ID: " + userID))
		})
	})
}
