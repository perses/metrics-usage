// Copyright 2024 The Perses Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metric

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/lithammer/fuzzysearch/fuzzy"
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

	ech.POST("/api/v1/invalid_metrics", e.PushMetricsUsage)
	ech.GET("/api/v1/invalid_metrics", e.ListInvalidMetrics)
	ech.GET("/api/v1/pending_usages", e.ListPendingUsages)
}

func (e *endpoint) GetMetric(ctx echo.Context) error {
	name := ctx.Param("id")
	metric := e.db.GetMetric(name)
	if metric == nil {
		return echo.NewHTTPError(http.StatusNotFound)
	}
	return ctx.JSON(http.StatusOK, metric)
}

type request struct {
	MetricName          string `query:"metric_name"`
	Used                *bool  `query:"used"`
	MergeInvalidMetrics bool   `query:"merge_invalid_metrics"`
}

func (r *request) filter(validMetricList map[string]*v1.Metric, invalidMetricList map[string]*v1.InvalidMetric) map[string]*v1.Metric {
	result := make(map[string]*v1.Metric)

	if r.MergeInvalidMetrics {
		for _, metric := range invalidMetricList {
			for metricName := range metric.MatchingMetrics {
				if m, ok := validMetricList[metricName]; ok {
					m.Usage = v1.MergeUsage(m.Usage, metric.Usage)
				}
			}
		}
	}

	if len(r.MetricName) == 0 && r.Used == nil {
		return validMetricList
	}
	for k, v := range validMetricList {
		if len(r.MetricName) == 0 || fuzzy.Match(r.MetricName, k) {
			if r.Used == nil {
				result[k] = v
			} else if *r.Used && validMetricList[k].Usage != nil {
				result[k] = v
			} else if !*r.Used && validMetricList[k].Usage == nil {
				result[k] = v
			}
		}
	}
	return result
}

func (e *endpoint) ListMetrics(ctx echo.Context) error {
	req := &request{}
	err := ctx.Bind(req)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, echo.Map{"message": err.Error()})
	}
	var invalidMetricList map[string]*v1.InvalidMetric
	if req.MergeInvalidMetrics {
		invalidMetricList, err = e.db.ListInvalidMetrics()
		if err != nil {
			return ctx.JSON(http.StatusInternalServerError, echo.Map{"message": err.Error()})
		}
	}
	metricList, err := e.db.ListMetrics()
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, echo.Map{"message": err.Error()})
	}
	return ctx.JSON(http.StatusOK, req.filter(metricList, invalidMetricList))
}

func (e *endpoint) PushMetricsUsage(ctx echo.Context) error {
	data := make(map[string]*v1.MetricUsage)
	if err := ctx.Bind(&data); err != nil {
		return ctx.JSON(http.StatusBadRequest, echo.Map{"message": err.Error()})
	}
	e.db.EnqueueUsage(data)
	return ctx.JSON(http.StatusAccepted, echo.Map{"message": "OK"})
}

func (e *endpoint) ListInvalidMetrics(ctx echo.Context) error {
	list, err := e.db.ListInvalidMetrics()
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, echo.Map{"message": err.Error()})
	}
	return ctx.JSON(http.StatusOK, list)
}

func (e *endpoint) PushInvalidMetricsUsage(ctx echo.Context) error {
	data := make(map[string]*v1.MetricUsage)
	if err := ctx.Bind(&data); err != nil {
		return ctx.JSON(http.StatusBadRequest, echo.Map{"message": err.Error()})
	}
	e.db.EnqueueInvalidMetricsUsage(data)
	return ctx.JSON(http.StatusAccepted, echo.Map{"message": "OK"})
}

func (e *endpoint) ListPendingUsages(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, e.db.ListPendingUsage())
}
