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

	"github.com/go-openapi/strfmt"
	grafanaapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/client/search"
	grafanaModels "github.com/grafana/grafana-openapi-client-go/models"
	"github.com/perses/common/async"
	"github.com/perses/metrics-usage/config"
	"github.com/perses/metrics-usage/database"
	"github.com/perses/metrics-usage/pkg/analyze/grafana"
	modelAPIV1 "github.com/perses/metrics-usage/pkg/api/v1"
	"github.com/perses/metrics-usage/pkg/client"
	"github.com/perses/metrics-usage/usageclient"
	"github.com/sirupsen/logrus"
)

func NewCollector(db database.Database, cfg config.GrafanaCollector) (async.SimpleTask, error) {
	httpClient, err := config.NewHTTPClient(cfg.HTTPClient)
	if err != nil {
		return nil, err
	}
	url := cfg.HTTPClient.URL.URL

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
	logger := logrus.StandardLogger().WithField("collector", "grafana")

	if cfg.PublicURL != nil {
		url = cfg.PublicURL.URL
	}
	return &grafanaCollector{
		grafanaURL:    url.String(),
		grafanaClient: grafanaClient,
		metricUsageClient: &usageclient.Client{
			DB:                db,
			MetricUsageClient: metricUsageClient,
			Logger:            logger,
		},
		logger: logrus.StandardLogger().WithField("collector", "grafana"),
	}, nil
}

type grafanaCollector struct {
	metricUsageClient *usageclient.Client
	grafanaURL        string
	grafanaClient     *grafanaapi.GrafanaHTTPAPI
	logger            *logrus.Entry
}

var _ async.SimpleTask = &grafanaCollector{}

func (c *grafanaCollector) Execute(ctx context.Context, _ context.CancelFunc) error {
	hits, err := c.collectAllDashboardUID(ctx)
	if err != nil {
		c.logger.WithError(err).Error("failed to collect dashboard UIDs")
		return nil
	}
	c.logger.Infof("collecting %d Grafana dashboards", len(hits))

	for _, h := range hits {
		dashboard, getErr := c.getDashboard(h.UID)
		if getErr != nil {
			c.logger.WithError(getErr).Errorf("failed to get dashboard %q with UID %q", h.Title, h.UID)
			continue
		}
		c.logger.Debugf("extracting metrics for the dashboard %s with UID %q", h.Title, h.UID)
		metrics, partialMetrics, errs := grafana.Analyze(dashboard)
		for _, logErr := range errs {
			logErr.Log(c.logger)
		}
		metricUsage := c.generateUsage(metrics, dashboard)
		partialMetricsUsage := c.generateUsage(partialMetrics, dashboard)
		c.logger.Infof("%d metrics usage has been collected for the dashboard %q with UID %q", len(metricUsage), h.Title, h.UID)
		c.logger.Infof("%d metrics containing regexp or variable has been collected for the dashboard %q with UID %q", len(partialMetricsUsage), h.Title, h.UID)
		c.metricUsageClient.SendUsage(metricUsage, partialMetricsUsage)
	}
	return nil
}

func (c *grafanaCollector) getDashboard(uid string) (*grafana.SimplifiedDashboard, error) {
	response, err := c.grafanaClient.Dashboards.GetDashboardByUID(uid)
	if err != nil {
		return nil, err
	}
	rowData, err := json.Marshal(response.Payload.Dashboard)
	if err != nil {
		return nil, err
	}
	result := &grafana.SimplifiedDashboard{}
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

func (c *grafanaCollector) generateUsage(metricNames modelAPIV1.Set[string], currentDashboard *grafana.SimplifiedDashboard) map[string]*modelAPIV1.MetricUsage {
	metricUsage := make(map[string]*modelAPIV1.MetricUsage)
	dashboardURL := fmt.Sprintf("%s/d/%s", c.grafanaURL, currentDashboard.UID)
	for metricName := range metricNames {
		if usage, ok := metricUsage[metricName]; ok {
			usage.Dashboards.Add(modelAPIV1.DashboardUsage{
				ID:   currentDashboard.UID,
				Name: currentDashboard.Title,
				URL:  dashboardURL,
			})
		} else {
			metricUsage[metricName] = &modelAPIV1.MetricUsage{
				Dashboards: modelAPIV1.NewSet(modelAPIV1.DashboardUsage{
					ID:   currentDashboard.UID,
					Name: currentDashboard.Title,
					URL:  dashboardURL,
				}),
			}
		}
	}
	return metricUsage
}

func (c *grafanaCollector) String() string {
	return "grafana collector"
}
