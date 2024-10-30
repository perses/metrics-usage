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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/perses/common/async"
	"github.com/perses/metrics-usage/config"
	"github.com/perses/metrics-usage/database"
	modelAPIV1 "github.com/perses/metrics-usage/pkg/api/v1"
	"github.com/perses/metrics-usage/pkg/client"
	"github.com/perses/metrics-usage/utils"
	"github.com/perses/metrics-usage/utils/prometheus"
	"github.com/perses/perses/go-sdk/prometheus/query"
	"github.com/perses/perses/go-sdk/prometheus/variable/promql"
	persesClientV1 "github.com/perses/perses/pkg/client/api/v1"
	persesClientConfig "github.com/perses/perses/pkg/client/config"
	v1 "github.com/perses/perses/pkg/model/api/v1"
	"github.com/perses/perses/pkg/model/api/v1/common"
	"github.com/perses/perses/pkg/model/api/v1/dashboard"
	"github.com/perses/perses/pkg/model/api/v1/variable"
	"github.com/sirupsen/logrus"
)

var variableReplacer = strings.NewReplacer(
	"$__interval", "5m",
	"$__interval_ms", "5m",
	"$__rate_interval", "15s",
	"$__range", "1d",
	"$__range_s", "15s",
	"$__range_ms", "15",
	"$__dashboard", "the_infamous_one",
	"$__project", "perses",
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
	}, nil
}

type persesCollector struct {
	async.SimpleTask
	persesClient      persesClientV1.DashboardInterface
	db                database.Database
	metricUsageClient client.Client
	persesURL         string
}

func (c *persesCollector) Execute(_ context.Context, _ context.CancelFunc) error {
	dashboards, err := c.persesClient.List("")
	if err != nil {
		logrus.WithError(err).Error("Failed to get dashboards")
		return nil
	}

	metricUsage := make(map[string]*modelAPIV1.MetricUsage)

	for _, dash := range dashboards {
		c.extractMetricUsageFromVariables(metricUsage, dash.Spec.Variables, dash)
		c.extractMetricUsageFromPanels(metricUsage, dash.Spec.Panels, dash)
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

func (c *persesCollector) extractMetricUsageFromPanels(metricUsage map[string]*modelAPIV1.MetricUsage, panels map[string]*v1.Panel, currentDashboard *v1.Dashboard) {
	for panelName, panel := range panels {
		for i, q := range panel.Spec.Queries {
			if q.Spec.Plugin.Kind != query.PluginKind {
				logrus.Debugf("In panel %q, skipping query number %d, with the type %q", panelName, i, q.Spec.Plugin.Kind)
				continue
			}
			spec, err := convertPluginSpecToTimeSeriesQuery(q.Spec.Plugin)
			if err != nil {
				logrus.WithError(err).Error("Failed to convert plugin spec to TimeSeriesQuery")
				continue
			}
			if len(spec.Query) == 0 {
				logrus.Debugf("No PromQL expression for the query %d in the panel %q for the dashboard '%s/%s'", i, panelName, currentDashboard.Metadata.Project, currentDashboard.Metadata.Name)
				continue
			}
			metrics, err := prometheus.ExtractMetricNamesFromPromQL(replaceVariables(spec.Query))
			if err != nil {
				logrus.WithError(err).Errorf("Failed to extract metric names from query %d in the panel %q for the dashboard '%s/%s'", i, panelName, currentDashboard.Metadata.Project, currentDashboard.Metadata.Name)
				continue
			}
			c.populateUsage(metricUsage, metrics, currentDashboard)
		}
	}
}

func (c *persesCollector) extractMetricUsageFromVariables(metricUsage map[string]*modelAPIV1.MetricUsage, variables []dashboard.Variable, currentDashboard *v1.Dashboard) {
	for _, v := range variables {
		if v.Kind != variable.KindList {
			continue
		}
		variableList, typeErr := v.Spec.(*dashboard.ListVariableSpec)
		if !typeErr {
			logrus.Errorf("variable spec is not of type ListVariableSpec but of type %T", v.Spec)
			continue
		}
		if variableList.Plugin.Kind != promql.PluginKind {
			logrus.Debugf("skipping this variable %q as it shouldn't contain any PromQL expression", variableList.Plugin.Kind)
			continue
		}
		spec, err := convertPluginSpecToPromQLVariable(variableList.Plugin)
		if err != nil {
			logrus.WithError(err).Error("Failed to convert plugin spec to PromQL variable")
			continue
		}
		metrics, err := prometheus.ExtractMetricNamesFromPromQL(replaceVariables(spec.Expr))
		if err != nil {
			logrus.WithError(err).Errorf("Failed to extract metric names from variable for the dashboard '%s/%s'", currentDashboard.Metadata.Project, currentDashboard.Metadata.Name)
			continue
		}
		c.populateUsage(metricUsage, metrics, currentDashboard)
	}
}

func (c *persesCollector) populateUsage(metricUsage map[string]*modelAPIV1.MetricUsage, metricNames []string, currentDashboard *v1.Dashboard) {
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
}

func (c *persesCollector) String() string {
	return "perses collector"
}

func replaceVariables(expr string) string {
	return variableReplacer.Replace(expr)
}

func convertPluginSpecToPromQLVariable(plugin common.Plugin) (promql.PluginSpec, error) {
	data, err := json.Marshal(plugin.Spec)
	if err != nil {
		return promql.PluginSpec{}, err
	}
	var result promql.PluginSpec
	err = json.Unmarshal(data, &result)
	return result, err
}

func convertPluginSpecToTimeSeriesQuery(plugin common.Plugin) (query.PluginSpec, error) {
	data, err := json.Marshal(plugin.Spec)
	if err != nil {
		return query.PluginSpec{}, err
	}
	var result query.PluginSpec
	err = json.Unmarshal(data, &result)
	return result, err
}
