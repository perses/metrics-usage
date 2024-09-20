package metric

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	persesEcho "github.com/perses/common/echo"
	"github.com/perses/metrics-usage/database"
	v1 "github.com/perses/metrics-usage/pkg/api/v1"
)

func NewAPI(db database.Database) persesEcho.Register {
	return &endpoint{
		db: db,
	}
}

type endpoint struct {
	db database.Database
}

func (e *endpoint) RegisterRoute(ech *echo.Echo) {
	path := "/api/v1/metrics"
	ech.POST(path, e.PushMetricsUsage)
	ech.GET(path, e.ListMetrics)
	ech.GET(fmt.Sprintf("%s/:id", path), e.GetMetric)
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

func (e *endpoint) PushMetricsUsage(ctx echo.Context) error {
	data := make(map[string]*v1.MetricUsage)
	if err := ctx.Bind(&data); err != nil {
		return ctx.JSON(http.StatusBadRequest, echo.Map{"message": err.Error()})
	}
	e.db.EnqueueUsage(data)
	return ctx.JSON(http.StatusAccepted, echo.Map{"message": "OK"})
}
