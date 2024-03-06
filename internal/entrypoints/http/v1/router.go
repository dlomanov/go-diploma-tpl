package v1

import (
	"github.com/dlomanov/go-diploma-tpl/internal/deps"
	"github.com/dlomanov/go-diploma-tpl/internal/entrypoints/http/middlewares"
	_ "github.com/dlomanov/go-diploma-tpl/internal/entrypoints/http/v1/docs"
	"github.com/dlomanov/go-diploma-tpl/internal/entrypoints/http/v1/endpoints"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter - HTTP API entrypoint
//
//	@title		gophermart API
//	@version	1.0
func NewRouter(r chi.Router, c *deps.Container) {
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	endpoints.UseSwagger(r, c)
	endpoints.UseAuthEndpoints(r, c)

	r.Group(func(r chi.Router) {
		r.Use(middlewares.Auth(c))
		endpoints.UseOrderEndpoints(r, c)
		endpoints.UseBalanceEndpoints(r, c)
	})
}
