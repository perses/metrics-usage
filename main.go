package main

import (
	"flag"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/perses/common/app"
	"github.com/perses/metrics-usage/database"
)

type endpoint struct {
	db database.Database
}

func (e *endpoint) RegisterRoute(ech *echo.Echo) {
	ech.GET("/api/v1/metrics", e.ListMetrics)
	ech.GET("/api/v1/metrics/:id", e.GetMetric)
}

func (e *endpoint) GetMetric(ctx echo.Context) error {
	name := ctx.Param("id")
	metric := e.db.GetMetric(name)
	if metric == nil {
		return echo.NewHTTPError(http.StatusNotFound)
	}
	return ctx.JSON(http.StatusOK, metric)
}

func (e *endpoint) ListMetrics(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, e.db.ListMetrics())
}

func main() {
	flag.Parse()
	runner := app.NewRunner().WithDefaultHTTPServer("metrics_usage")

	db := database.New()

	runner.HTTPServerBuilder().APIRegistration(&endpoint{db: db})
	runner.Start()
}
