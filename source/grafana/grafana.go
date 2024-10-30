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

package grafana

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-openapi/strfmt"
	grafanaapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/client/search"
	grafanaModels "github.com/grafana/grafana-openapi-client-go/models"
	"github.com/perses/common/async"
	"github.com/perses/metrics-usage/config"
	"github.com/perses/metrics-usage/database"
	modelAPIV1 "github.com/perses/metrics-usage/pkg/api/v1"
	"github.com/perses/metrics-usage/pkg/client"
	"github.com/perses/metrics-usage/utils"
	"github.com/perses/metrics-usage/utils/prometheus"
	"github.com/sirupsen/logrus"
)

var variableReplacer = strings.NewReplacer(
	"$__interval", "5m",
	"$interval", "5m",
	"$resolution", "5m",
	"$__rate_interval", "15s",
	"$rate_interval", "15s",
	"$__range", "1d",
	"${__range_s:glob}", "15",
	"${__range_s}", "15",
)

func NewCollector(db database.Database, cfg config.GrafanaCollector) (async.SimpleTask, error) {
	httpClient, err := config.NewHTTPClient(cfg.HTTPClient)
	url := cfg.HTTPClient.URL.URL
	if err != nil {
		return nil, err
	}
	var metricUsageClient client.Client
	if cfg.MetricUsageClient != nil {
		metricUsageClient, err = client.New(*cfg.MetricUsageClient)
		if err != nil {
			return nil, err
		}
	}
	transportConfig := &grafanaapi.TransportConfig{
		Host:     url.Host,
		BasePath: grafanaapi.DefaultBasePath,
		Schemes:  []string{url.Scheme},
		Client:   httpClient,
	}
	grafanaClient := grafanaapi.NewHTTPClientWithConfig(strfmt.Default, transportConfig)
	return &grafanaCollector{
		db:                db,
		grafanaURL:        url.String(),
		grafanaClient:     grafanaClient,
		metricUsageClient: metricUsageClient,
	}, nil
}

type grafanaCollector struct {
	async.SimpleTask
	db                database.Database
	metricUsageClient client.Client
	grafanaURL        string
	grafanaClient     *grafanaapi.GrafanaHTTPAPI
}

func (c *grafanaCollector) Execute(ctx context.Context, _ context.CancelFunc) error {
	hits, err := c.collectAllDashboardUID(ctx)
	if err != nil {
		logrus.WithError(err).Error("failed to collect dashboard UIDs")
		return nil
	}

	metricUsage := make(map[string]*modelAPIV1.MetricUsage)
	for _, h := range hits {
		dashboard, getErr := c.getDashboard(h.UID)
		if getErr != nil {
			logrus.WithError(getErr).Errorf("failed to get dashboard %q with UID %q", h.Title, h.UID)
			continue
		}
		c.extractMetricUsage(metricUsage, dashboard)
	}
	if len(metricUsage) > 0 {
		if c.metricUsageClient != nil {
			// In this case, that means we have to send the data to a remote server.
			if sendErr := c.metricUsageClient.Usage(metricUsage); sendErr != nil {
				logrus.WithError(sendErr).Error("Failed to send usage metric")
			}
		} else {
			c.db.EnqueueUsage(metricUsage)
		}
	}
	return nil
}

func (c *grafanaCollector) extractMetricUsage(metricUsage map[string]*modelAPIV1.MetricUsage, dashboard *simplifiedDashboard) {
	c.extractMetricUsageFromPanels(metricUsage, dashboard.Panels, dashboard)
	for _, r := range dashboard.Rows {
		c.extractMetricUsageFromPanels(metricUsage, r.Panels, dashboard)
		// TODO extract metric usage from variable
	}
}

func (c *grafanaCollector) extractMetricUsageFromPanels(metricUsage map[string]*modelAPIV1.MetricUsage, panels []panel, dashboard *simplifiedDashboard) {
	for _, p := range panels {
		for _, t := range extractTarget(p) {
			if len(t.Expr) == 0 {
				continue
			}
			metrics, err := prometheus.ExtractMetricNamesFromPromQL(replaceVariables(t.Expr))
			if err != nil {
				logrus.WithError(err).Errorf("failed to extract metric names from PromQL expression in the panel %q for the dashboard %s/%s", p.Title, dashboard.Title, dashboard.UID)
			}
			c.populateUsage(metricUsage, metrics, dashboard)
		}
	}
}

func (c *grafanaCollector) getDashboard(uid string) (*simplifiedDashboard, error) {
	response, err := c.grafanaClient.Dashboards.GetDashboardByUID(uid)
	if err != nil {
		return nil, err
	}
	rowData, err := json.Marshal(response.Payload.Dashboard)
	if err != nil {
		return nil, err
	}
	result := &simplifiedDashboard{}
	return result, json.Unmarshal(rowData, &result)
}

func (c *grafanaCollector) collectAllDashboardUID(ctx context.Context) ([]*grafanaModels.Hit, error) {
	var currentPage int64 = 1
	var result []*grafanaModels.Hit
	searchOk := true
	// value based on the comment from the code here: https://github.com/grafana/grafana-openapi-client-go/blob/9d96c2007bd8c89981630106307c8764e3d02747/client/search/search_parameters.go#L151
	searchType := "dash-db"

	for searchOk {
		nextPageResult, err := c.grafanaClient.Search.Search(&search.SearchParams{
			Context: ctx,
			Type:    &searchType,
			Page:    &currentPage,
		})
		if err != nil {
			return nil, err
		}
		searchOk = nextPageResult.IsSuccess() && len(nextPageResult.Payload) > 0
		currentPage++
		if searchOk {
			result = append(result, nextPageResult.Payload...)
		}
	}
	return result, nil
}

func (c *grafanaCollector) populateUsage(metricUsage map[string]*modelAPIV1.MetricUsage, metricNames []string, currentDashboard *simplifiedDashboard) {
	dashboardURL := fmt.Sprintf("%s/d/%s", c.grafanaURL, currentDashboard.UID)
	for _, metricName := range metricNames {
		if usage, ok := metricUsage[metricName]; ok {
			usage.Dashboards = utils.InsertIfNotPresent(usage.Dashboards, dashboardURL)
		} else {
			metricUsage[metricName] = &modelAPIV1.MetricUsage{
				Dashboards: []string{dashboardURL},
			}
		}
	}
}

func (c *grafanaCollector) String() string {
	return "grafana collector"
}

func replaceVariables(expr string) string {
	return variableReplacer.Replace(expr)
}
