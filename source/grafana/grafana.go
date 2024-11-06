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
	"regexp"
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

var (
	labelValuesRegexp            = regexp.MustCompile(`(?s)label_values\((.+),.+\)`)
	labelValuesNoQueryRegexp     = regexp.MustCompile(`(?s)label_values\((.+)\)`)
	queryResultRegexp            = regexp.MustCompile(`(?s)query_result\((.+)\)`)
	variableRangeQueryRangeRegex = regexp.MustCompile(`\[\$?\w+?]`)
	variableSubqueryRangeRegex   = regexp.MustCompile(`\[\$?\w+:\$?\w+?]`)
	globalVariableList           = map[string]string{
		"__interval":          "20m",
		"interval":            "20m",
		"__interval_ms":       "1200000",
		"resolution":          "5m",
		"__rate_interval":     "20m",
		"rate_interval":       "20m",
		"__rate_interval_ms":  "1200000",
		"__range":             "1d",
		"__range_s:glob":      "15",
		"__range_s":           "15",
		"__range_ms":          "15",
		"__from":              "1594671549254",
		"__from:date":         "2020-07-13T20:19:09.254Z",
		"__from:date:iso":     "2020-07-13T20:19:09.254Z",
		"__from:date:seconds": "1594671549",
		"__from:date:YYYY-MM": "2020-07",
		"__to":                "1594671549254",
		"__to:date":           "2020-07-13T20:19:09.254Z",
		"__to:date:iso":       "2020-07-13T20:19:09.254Z",
		"__to:date:seconds":   "1594671549",
		"__to:date:YYYY-MM":   "2020-07",
		"__user":              "foo",
		"__org":               "perses",
		"__name":              "john",
		"__dashboard":         "the_infamous_one",
	}
	variableReplacer = strings.NewReplacer(generateGrafanaVariableSyntaxReplacer(globalVariableList)...)
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
		logger:            logrus.StandardLogger().WithField("collector", "grafana"),
	}, nil
}

type logError struct {
	msg string
	err error
}

type grafanaCollector struct {
	async.SimpleTask
	db                database.Database
	metricUsageClient client.Client
	grafanaURL        string
	grafanaClient     *grafanaapi.GrafanaHTTPAPI
	logger            *logrus.Entry
}

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
		metrics, errs := extractMetrics(dashboard)
		for _, extractError := range errs {
			c.logger.WithError(extractError.err).Error(extractError.msg)
		}
		metricUsage := c.generateUsage(metrics, dashboard)
		c.logger.Infof("%d metrics usage has been collected for the dashboard %q with UID %q", len(metricUsage), h.Title, h.UID)
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

func (c *grafanaCollector) generateUsage(metricNames []string, currentDashboard *simplifiedDashboard) map[string]*modelAPIV1.MetricUsage {
	metricUsage := make(map[string]*modelAPIV1.MetricUsage)
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
	return metricUsage
}

func (c *grafanaCollector) String() string {
	return "grafana collector"
}

func replaceVariables(expr string, staticVariables *strings.Replacer) string {
	newExpr := staticVariables.Replace(expr)
	newExpr = variableReplacer.Replace(newExpr)
	newExpr = variableRangeQueryRangeRegex.ReplaceAllLiteralString(newExpr, `[5m]`)
	newExpr = variableSubqueryRangeRegex.ReplaceAllLiteralString(newExpr, `[5m:1m]`)
	return newExpr
}

func generateGrafanaVariableSyntaxReplacer(variables map[string]string) []string {
	var result []string
	for variable, value := range variables {
		result = append(result, fmt.Sprintf("$%s", variable), value, fmt.Sprintf("${%s}", variable), value)
	}
	return result
}

func extractMetrics(dashboard *simplifiedDashboard) ([]string, []logError) {
	staticVariables := strings.NewReplacer(generateGrafanaVariableSyntaxReplacer(extractStaticVariables(dashboard.Templating.List))...)
	m1, err1 := extractMetricsFromPanels(dashboard.Panels, staticVariables, dashboard)
	for _, r := range dashboard.Rows {
		m2, err2 := extractMetricsFromPanels(r.Panels, staticVariables, dashboard)
		m1 = utils.Merge(m1, m2)
		err1 = append(err1, err2...)
	}
	m3, err3 := extractMetricsFromVariables(dashboard.Templating.List, staticVariables, dashboard)
	return utils.Merge(m1, m3), append(err1, err3...)
}

func extractMetricsFromPanels(panels []panel, staticVariables *strings.Replacer, dashboard *simplifiedDashboard) ([]string, []logError) {
	var errs []logError
	var result []string
	for _, p := range panels {
		for _, t := range extractTarget(p) {
			if len(t.Expr) == 0 {
				continue
			}
			metrics, err := prometheus.ExtractMetricNamesFromPromQL(replaceVariables(t.Expr, staticVariables))
			if err != nil {
				errs = append(errs, logError{
					err: err,
					msg: fmt.Sprintf("failed to extract metric names from PromQL expression in the panel %q for the dashboard %s/%s", p.Title, dashboard.Title, dashboard.UID),
				})
			} else {
				result = utils.Merge(result, metrics)
			}
		}
	}
	return result, errs
}

func extractMetricsFromVariables(variables []templateVar, staticVariables *strings.Replacer, dashboard *simplifiedDashboard) ([]string, []logError) {
	var errs []logError
	var result []string
	for _, v := range variables {
		if v.Type != "query" {
			continue
		}
		query, err := v.extractQueryFromVariableTemplating()
		if err != nil {
			errs = append(errs, logError{
				err: err,
				msg: fmt.Sprintf("failed to extract query in variable %q for the dashboard %s/%s", v.Name, dashboard.Title, dashboard.UID),
			})
			continue
		}
		// label_values(query, label)
		if labelValuesRegexp.MatchString(query) {
			sm := labelValuesRegexp.FindStringSubmatch(query)
			if len(sm) > 0 {
				query = sm[1]
			} else {
				continue
			}
		} else if labelValuesNoQueryRegexp.MatchString(query) {
			// No query so no metric.
			continue
		} else if queryResultRegexp.MatchString(query) {
			// query_result(query)
			query = queryResultRegexp.FindStringSubmatch(query)[1]
		}
		metrics, err := prometheus.ExtractMetricNamesFromPromQL(replaceVariables(query, staticVariables))
		if err != nil {
			errs = append(errs, logError{
				err: err,
				msg: fmt.Sprintf("failed to extract metric names from PromQL expression in variable %q for the dashboard %s/%s", v.Name, dashboard.Title, dashboard.UID),
			})
		} else {
			result = utils.Merge(result, metrics)
		}
	}
	return result, errs
}

func extractStaticVariables(variables []templateVar) map[string]string {
	result := make(map[string]string)
	for _, v := range variables {
		if v.Type == "query" {
			// We don't want to look at the runtime query. We are using them to extract metrics instead.
			continue
		}
		if len(v.Options) > 0 {
			result[v.Name] = v.Options[0].Value
		}
	}
	return result
}
