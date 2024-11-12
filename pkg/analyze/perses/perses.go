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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/perses/metrics-usage/pkg/analyze/prometheus"
	modelAPIV1 "github.com/perses/metrics-usage/pkg/api/v1"
	"github.com/perses/metrics-usage/utils"
	"github.com/perses/perses/go-sdk/prometheus/query"
	"github.com/perses/perses/go-sdk/prometheus/variable/promql"
	v1 "github.com/perses/perses/pkg/model/api/v1"
	"github.com/perses/perses/pkg/model/api/v1/common"
	"github.com/perses/perses/pkg/model/api/v1/dashboard"
	"github.com/perses/perses/pkg/model/api/v1/variable"
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

func Analyze(dashboard *v1.Dashboard) ([]string, []*modelAPIV1.LogError) {
	m1, err1 := extractMetricUsageFromVariables(dashboard.Spec.Variables, dashboard)
	m2, err2 := extractMetricUsageFromPanels(dashboard.Spec.Panels, dashboard)
	return utils.Merge(m1, m2), append(err1, err2...)
}

func extractMetricUsageFromPanels(panels map[string]*v1.Panel, currentDashboard *v1.Dashboard) ([]string, []*modelAPIV1.LogError) {
	var errs []*modelAPIV1.LogError
	var result []string
	for panelName, panel := range panels {
		for i, q := range panel.Spec.Queries {
			if q.Spec.Plugin.Kind != query.PluginKind {
				continue
			}
			spec, err := convertPluginSpecToTimeSeriesQuery(q.Spec.Plugin)
			if err != nil {
				errs = append(errs, &modelAPIV1.LogError{
					Error:   err,
					Message: "Failed to convert plugin spec to TimeSeriesQuery",
				})
				continue
			}
			if len(spec.Query) == 0 {
				// No PromQL expression for the query
				continue
			}
			metrics, err := prometheus.AnalyzePromQLExpression(replaceVariables(spec.Query))
			if err != nil {
				errs = append(errs, &modelAPIV1.LogError{
					Error:   err,
					Message: fmt.Sprintf("Failed to extract metric names from query %d in the panel %q for the dashboard '%s/%s'", i, panelName, currentDashboard.Metadata.Project, currentDashboard.Metadata.Name),
				})
				continue
			}
			result = utils.Merge(result, metrics)
		}
	}
	return result, errs
}

func extractMetricUsageFromVariables(variables []dashboard.Variable, currentDashboard *v1.Dashboard) ([]string, []*modelAPIV1.LogError) {
	var errs []*modelAPIV1.LogError
	var result []string
	for _, v := range variables {
		if v.Kind != variable.KindList {
			continue
		}
		variableList, typeErr := v.Spec.(*dashboard.ListVariableSpec)
		if !typeErr {
			errs = append(errs, &modelAPIV1.LogError{
				Error: fmt.Errorf("variable spec is not of type ListVariableSpec but of type %T", v.Spec),
			})
			continue
		}
		if variableList.Plugin.Kind != promql.PluginKind {
			// Skipping this variable as it shouldn't contain any PromQL expression.
			continue
		}
		spec, err := convertPluginSpecToPromQLVariable(variableList.Plugin)
		if err != nil {
			errs = append(errs, &modelAPIV1.LogError{
				Error:   err,
				Message: "Failed to convert plugin spec to PromQL variable",
			})
			continue
		}
		metrics, err := prometheus.AnalyzePromQLExpression(replaceVariables(spec.Expr))
		if err != nil {
			errs = append(errs, &modelAPIV1.LogError{
				Error:   err,
				Message: fmt.Sprintf("Failed to extract metric names from variable for the dashboard '%s/%s'", currentDashboard.Metadata.Project, currentDashboard.Metadata.Name),
			})
			continue
		}
		result = utils.Merge(result, metrics)
	}
	return result, errs
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
