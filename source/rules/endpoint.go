package rules

import (
	"net/http"

	"github.com/labstack/echo/v4"
	persesEcho "github.com/perses/common/echo"
	"github.com/perses/metrics-usage/database"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

func NewAPI(db database.Database) persesEcho.Register {
	return &endpoint{
		db: db,
	}
}

type request struct {
	Source string         `json:"source"`
	Groups []v1.RuleGroup `json:"groups"`
}

type endpoint struct {
	db database.Database
}

func (e *endpoint) RegisterRoute(ech *echo.Echo) {
	path := "/api/v1/rules"
	ech.POST(path, e.PushRules)
}

func (e *endpoint) PushRules(ctx echo.Context) error {
	var data request
	if err := ctx.Bind(&data); err != nil {
		return ctx.JSON(http.StatusBadRequest, echo.Map{"message": err.Error()})
	}
	metricUsage := extractMetricUsageFromRules(data.Groups, data.Source)
	if len(metricUsage) > 0 {
		e.db.EnqueueUsage(metricUsage)
	}
	return ctx.JSON(http.StatusAccepted, echo.Map{"message": "OK"})
}
