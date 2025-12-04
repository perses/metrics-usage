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
	"fmt"
	"regexp"
	"strings"

	"github.com/perses/metrics-usage/pkg/analyze/expr"
	"github.com/perses/metrics-usage/pkg/analyze/parser"
	"github.com/perses/metrics-usage/pkg/analyze/prometheus"
	modelAPIV1 "github.com/perses/metrics-usage/pkg/api/v1"
)

type variableTuple struct {
	name  string
	value string
}

var (
	labelValuesRegexp            = regexp.MustCompile(`(?s)label_values\((.+),.+\)`)
	labelValuesNoQueryRegexp     = regexp.MustCompile(`(?s)label_values\((.+)\)`)
	queryResultRegexp            = regexp.MustCompile(`(?s)query_result\((.+)\)`)
	metricsRegexp                = regexp.MustCompile(`(?s)metrics\((.+)\)`)
	variableRangeQueryRangeRegex = regexp.MustCompile(`\[\$?\w+?]`)
	variableSubqueryRangeRegex   = regexp.MustCompile(`\[\$?\w+:\$?\w+?]`)
	globalVariableList           = []variableTuple{
		// Don't change the order.
		// The order matters because, when replacing the variable with its value in the expression, if, for example,
		// __interval is replaced before __interval_ms, then you might have partially replaced the variable.
		// Example: 1 / __interval_ms will give 1 / 20m_s which is not a correct PromQL expression.
		// So we need to replace __interval_ms before __interval.
		// Same thing applied for every variable starting with the same prefix. Like __from, __to.
		{
			name:  "__interval_ms",
			value: "1200000",
		},
		{
			name:  "__interval",
			value: "20m",
		},
		{
			name:  "interval",
			value: "20m",
		},
		{
			name:  "resolution",
			value: "5m",
		},
		{
			name:  "__rate_interval_ms",
			value: "1200000",
		},
		{
			name:  "__rate_interval",
			value: "20m",
		},
		{
			name:  "rate_interval",
			value: "20m",
		},
		{
			name:  "__range_s:glob",
			value: "15",
		},
		{
			name:  "__range_s",
			value: "15",
		},
		{
			name:  "__range_ms",
			value: "15",
		},
		{
			name:  "__range",
			value: "1d",
		},
		{
			name:  "__from:date:YYYY-MM",
			value: "2020-07",
		},
		{
			name:  "__from:date:seconds",
			value: "1594671549",
		},
		{
			name:  "__from:date:iso",
			value: "2020-07-13T20:19:09.254Z",
		},
		{
			name:  "__from:date",
			value: "2020-07-13T20:19:09.254Z",
		},
		{
			name:  "__from",
			value: "1594671549254",
		},
		{
			name:  "__to:date:YYYY-MM",
			value: "2020-07",
		},
		{
			name:  "__to:date:seconds",
			value: "1594671549",
		},
		{
			name:  "__to:date:iso",
			value: "2020-07-13T20:19:09.254Z",
		},
		{
			name:  "__to:date",
			value: "2020-07-13T20:19:09.254Z",
		},
		{
			name:  "__to",
			value: "1594671549254",
		},
		{
			name:  "__user",
			value: "foo",
		},
		{
			name:  "__org",
			value: "perses",
		},
		{
			name:  "__name",
			value: "john",
		},
		{
			name:  "__dashboard",
			value: "the_infamous_one",
		},
	}
	variableReplacer = strings.NewReplacer(generateGrafanaTupleVariableSyntaxReplacer(globalVariableList)...)
)

func Analyze(dashboard *SimplifiedDashboard, analyzer expr.Analyzer) (modelAPIV1.Set[string], modelAPIV1.Set[string], []*modelAPIV1.LogError) {
	if analyzer == nil {
		return nil, nil, []*modelAPIV1.LogError{
			{Error: fmt.Errorf("expression analyzer is not configured")},
		}
	}
	staticVariables := strings.NewReplacer(generateGrafanaVariableSyntaxReplacer(extractStaticVariables(dashboard.Templating.List))...)
	allVariableNames := collectAllVariableName(dashboard.Templating.List)
	m1, inv1, err1 := extractMetricsFromPanels(dashboard.Panels, staticVariables, allVariableNames, dashboard, analyzer)
	for _, r := range dashboard.Rows {
		m2, inv2, err2 := extractMetricsFromPanels(r.Panels, staticVariables, allVariableNames, dashboard, analyzer)
		m1.Merge(m2)
		inv1.Merge(inv2)
		err1 = append(err1, err2...)
	}
	m3, inv3, err3 := extractMetricsFromVariables(dashboard.Templating.List, staticVariables, allVariableNames, dashboard, analyzer)
	m1.Merge(m3)
	inv1.Merge(inv3)
	return m1, inv1, append(err1, err3...)
}

func extractMetricsFromPanels(panels []Panel, staticVariables *strings.Replacer, allVariableNames modelAPIV1.Set[string], dashboard *SimplifiedDashboard, analyzer expr.Analyzer) (modelAPIV1.Set[string], modelAPIV1.Set[string], []*modelAPIV1.LogError) {
	var errs []*modelAPIV1.LogError
	result := modelAPIV1.Set[string]{}
	partialMetricsResult := modelAPIV1.Set[string]{}
	for _, p := range panels {
		for _, t := range extractTarget(p) {
			if len(t.Expr) == 0 {
				continue
			}
			exprWithVariableReplaced := replaceVariables(t.Expr, staticVariables)
			metrics, partialMetrics, err := analyzer.Analyze(exprWithVariableReplaced)
			if err != nil {
				otherMetrics := parser.ExtractMetricNameWithVariable(exprWithVariableReplaced)
				if len(otherMetrics) > 0 {
					for m := range otherMetrics {
						if prometheus.IsValidMetricName(m) {
							result.Add(m)
						} else {
							partialMetricsResult.Add(formatVariableInMetricName(m, allVariableNames))
						}
					}
				} else {
					errs = append(errs, &modelAPIV1.LogError{
						Error:   err,
						Message: fmt.Sprintf("failed to extract metric names from PromQL expression in the panel %q for the dashboard %s/%s", p.Title, dashboard.Title, dashboard.UID),
					})
				}
			} else {
				result.Merge(metrics)
				partialMetricsResult.Merge(partialMetrics)
			}
		}
	}
	return result, partialMetricsResult, errs
}

func extractMetricsFromVariables(variables []templateVar, staticVariables *strings.Replacer, allVariableNames modelAPIV1.Set[string], dashboard *SimplifiedDashboard, analyzer expr.Analyzer) (modelAPIV1.Set[string], modelAPIV1.Set[string], []*modelAPIV1.LogError) {
	var errs []*modelAPIV1.LogError
	result := modelAPIV1.Set[string]{}
	partialMetricsResult := modelAPIV1.Set[string]{}
	for _, v := range variables {
		if v.Type != "query" {
			continue
		}
		query, err := v.extractQueryFromVariableTemplating()
		if err != nil {
			// It appears when there is an issue, we cannot do anything about it,
			// and usually the variable is not the one we are looking for.
			// So we just log it as a warning
			errs = append(errs, &modelAPIV1.LogError{
				Warning: err,
				Message: fmt.Sprintf("failed to extract query in variable %q for the dashboard %s/%s", v.Name, dashboard.Title, dashboard.UID),
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
			// metrics(.*partial_metric_name)
		} else if metricsRegexp.MatchString(query) {
			// for this particular use case, the query is a partial metric names so there is no need to use the PromQL parser.
			query = metricsRegexp.FindStringSubmatch(query)[1]
			partialMetricsResult.Add(formatVariableInMetricName(query, allVariableNames))
			continue
		}
		exprWithVariableReplaced := replaceVariables(query, staticVariables)
		metrics, partialMetrics, err := analyzer.Analyze(exprWithVariableReplaced)
		if err != nil {
			otherMetrics := parser.ExtractMetricNameWithVariable(exprWithVariableReplaced)
			if len(otherMetrics) > 0 {
				for m := range otherMetrics {
					if prometheus.IsValidMetricName(m) {
						result.Add(m)
					} else {
						partialMetricsResult.Add(formatVariableInMetricName(m, allVariableNames))
					}
				}
			} else {
				errs = append(errs, &modelAPIV1.LogError{
					Error:   err,
					Message: fmt.Sprintf("failed to extract metric names from PromQL expression in variable %q for the dashboard %s/%s", v.Name, dashboard.Title, dashboard.UID),
				})
			}
		} else {
			result.Merge(metrics)
			partialMetricsResult.Merge(partialMetrics)
		}
	}
	return result, partialMetricsResult, errs
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
			if v.Type == "custom" {
				// It seems the variable format <variable:value> ca be used for the "custom" variables.
				result[fmt.Sprintf("%s:value", v.Name)] = v.Options[0].Value
			}
		}
	}
	return result
}

func collectAllVariableName(variables []templateVar) modelAPIV1.Set[string] {
	result := modelAPIV1.Set[string]{}
	for _, v := range variables {
		result.Add(v.Name)
	}
	return result
}

func replaceVariables(expr string, staticVariables *strings.Replacer) string {
	newExpr := staticVariables.Replace(expr)
	newExpr = variableReplacer.Replace(newExpr)
	newExpr = variableRangeQueryRangeRegex.ReplaceAllLiteralString(newExpr, `[5m]`)
	newExpr = variableSubqueryRangeRegex.ReplaceAllLiteralString(newExpr, `[5m:1m]`)
	return newExpr
}

// formatVariableInMetricName will replace the syntax of the variable by another one that can actually be parsed.
// It will be useful for later when we want to know which metrics, this metric with variable is covered.
func formatVariableInMetricName(metric string, variables modelAPIV1.Set[string]) string {
	for v := range variables {
		metric = strings.ReplaceAll(metric, fmt.Sprintf("$%s", v), fmt.Sprintf("${%s}", v))
	}
	return metric
}

func generateGrafanaVariableSyntaxReplacer(variables map[string]string) []string {
	var result []string
	for variable, value := range variables {
		result = append(result, fmt.Sprintf("$%s", variable), value, fmt.Sprintf("${%s}", variable), value)
	}
	return result
}

func generateGrafanaTupleVariableSyntaxReplacer(variables []variableTuple) []string {
	var result []string
	for _, v := range variables {
		result = append(result, fmt.Sprintf("$%s", v.name), v.value, fmt.Sprintf("${%s}", v.name), v.value)
	}
	return result
}
