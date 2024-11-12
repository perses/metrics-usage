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

package perses

import (
	"context"
	"fmt"

	"github.com/perses/common/async"
	"github.com/perses/metrics-usage/config"
	"github.com/perses/metrics-usage/database"
	"github.com/perses/metrics-usage/pkg/analyze/perses"
	modelAPIV1 "github.com/perses/metrics-usage/pkg/api/v1"
	"github.com/perses/metrics-usage/pkg/client"
	"github.com/perses/metrics-usage/utils"
	persesClientV1 "github.com/perses/perses/pkg/client/api/v1"
	persesClientConfig "github.com/perses/perses/pkg/client/config"
	v1 "github.com/perses/perses/pkg/model/api/v1"
	"github.com/sirupsen/logrus"
)

func NewCollector(db database.Database, cfg config.PersesCollector) (async.SimpleTask, error) {
	restClient, err := persesClientConfig.NewRESTClient(cfg.HTTPClient)
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
	return &persesCollector{
		SimpleTask:        nil,
		persesClient:      persesClientV1.NewWithClient(restClient).Dashboard(""),
		db:                db,
		metricUsageClient: metricUsageClient,
		persesURL:         cfg.HTTPClient.URL.String(),
		logger:            logrus.StandardLogger().WithField("collector", "perses"),
	}, nil
}

type persesCollector struct {
	async.SimpleTask
	persesClient      persesClientV1.DashboardInterface
	db                database.Database
	metricUsageClient client.Client
	persesURL         string
	logger            *logrus.Entry
}

func (c *persesCollector) Execute(_ context.Context, _ context.CancelFunc) error {
	dashboards, err := c.persesClient.List("")
	if err != nil {
		c.logger.WithError(err).Error("Failed to get dashboards")
		return nil
	}

	for _, dash := range dashboards {
		metrics, errs := perses.Analyze(dash)
		for _, logErr := range errs {
			logErr.Log(c.logger)
		}
		metricUsage := c.generateUsage(metrics, dash)
		c.logger.Infof("%d metrics usage has been collected for the dashboard %s/%s", len(metricUsage), dash.Metadata.Project, dash.Metadata.Name)
		if len(metricUsage) > 0 {
			if c.metricUsageClient != nil {
				// In this case, that means we have to send the data to a remote server.
				if sendErr := c.metricUsageClient.Usage(metricUsage); sendErr != nil {
					c.logger.WithError(sendErr).Error("Failed to send usage metric")
				}
			} else {
				c.db.EnqueueUsage(metricUsage)
			}
		}
	}
	return nil
}

func (c *persesCollector) generateUsage(metricNames []string, currentDashboard *v1.Dashboard) map[string]*modelAPIV1.MetricUsage {
	metricUsage := make(map[string]*modelAPIV1.MetricUsage)
	dashboardURL := fmt.Sprintf("%s/api/v1/projects/%s/dashboards/%s", c.persesURL, currentDashboard.Metadata.Project, currentDashboard.Metadata.Name)
	for _, metricName := range metricNames {
		if usage, ok := metricUsage[metricName]; ok {
			usage.Dashboards = utils.InsertIfNotPresent(usage.Dashboards, dashboardURL)
		} else {
			metricUsage[metricName] = &modelAPIV1.MetricUsage{
				Dashboards: []string{dashboardURL},
			}
		}
	}
	return metricUsage
}

func (c *persesCollector) String() string {
	return "perses collector"
}
