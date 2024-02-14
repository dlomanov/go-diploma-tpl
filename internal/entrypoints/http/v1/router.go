package v1

import (
	"github.com/dlomanov/go-diploma-tpl/internal/deps"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(r chi.Router, c *deps.Container) {
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	UseAuthEndpoints(r, c)
}
