package endpoints

import (
	"github.com/dlomanov/go-diploma-tpl/internal/deps"
	"github.com/go-chi/chi/v5"
	"github.com/swaggo/http-swagger/v2"

	"strings"
)

func UseSwagger(router chi.Router, c *deps.Container) {
	url := c.Config.RunAddress
	if !strings.HasPrefix(url, "http") {
		url = "http://" + url
	}

	router.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL(url+"/swagger/doc.json"),
	))

	c.Logger.Debug("swagger endpoint registered " + url + "/swagger/index.html")
}
