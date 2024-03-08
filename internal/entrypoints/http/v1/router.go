package v1

import (
	"github.com/dlomanov/go-diploma-tpl/internal/entrypoints/http/middlewares"
	_ "github.com/dlomanov/go-diploma-tpl/internal/entrypoints/http/v1/docs"
	"github.com/dlomanov/go-diploma-tpl/internal/entrypoints/http/v1/endpoints"
	"github.com/dlomanov/go-diploma-tpl/internal/infra/deps"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter - HTTP AccrualAPI entrypoint
//
//	@title		gophermart AccrualAPI
//	@version	1.0
func NewRouter(r chi.Router, c *deps.Container) {
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	endpoints.UseSwagger(r, c)
	endpoints.UseAuthEndpoints(r, c)

	middleware.AllowContentType()

	r.Group(func(r chi.Router) {
		r.Use(middlewares.Auth(c))
		endpoints.UseOrderEndpoints(r, c)
		endpoints.UseBalanceEndpoints(r, c)
	})
}
