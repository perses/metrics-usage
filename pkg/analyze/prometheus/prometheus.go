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

package prometheus

import (
	"fmt"
	"regexp"

	"github.com/perses/common/set"
	"github.com/perses/metrics-usage/pkg/analyze/expr"
	modelAPIV1 "github.com/perses/metrics-usage/pkg/api/v1"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

var validMetricName = regexp.MustCompile(`^[a-zA-Z_:][a-zA-Z0-9_:]*$`)

func Analyze(ruleGroups []v1.RuleGroup, source string, analyzer expr.Analyzer) (map[string]*modelAPIV1.MetricUsage, map[string]*modelAPIV1.MetricUsage, []*modelAPIV1.LogError) {
	var errs []*modelAPIV1.LogError
	metricUsage := make(map[string]*modelAPIV1.MetricUsage)
	partialMetricUsage := make(map[string]*modelAPIV1.MetricUsage)
	for _, ruleGroup := range ruleGroups {
		for _, rule := range ruleGroup.Rules {
			switch v := rule.(type) {
			case v1.RecordingRule:
				metricNames, partialMetrics, parserErr := AnalyzePromQLExpression(v.Query, analyzer)
				if parserErr != nil {
					errs = append(errs, &modelAPIV1.LogError{
						Message: fmt.Sprintf("Failed to extract metric name for the ruleGroup %q and the recordingRule %q", ruleGroup.Name, v.Name),
						Error:   parserErr,
					})
					continue
				}
				populateUsage(metricUsage,
					metricNames,
					modelAPIV1.RuleUsage{
						PromLink:   source,
						GroupName:  ruleGroup.Name,
						Name:       v.Name,
						Expression: v.Query,
					},
					false,
				)
				populateUsage(partialMetricUsage,
					partialMetrics,
					modelAPIV1.RuleUsage{
						PromLink:   source,
						GroupName:  ruleGroup.Name,
						Name:       v.Name,
						Expression: v.Query,
					},
					false,
				)
			case v1.AlertingRule:
				metricNames, partialMetrics, parserErr := AnalyzePromQLExpression(v.Query, analyzer)
				if parserErr != nil {
					errs = append(errs, &modelAPIV1.LogError{
						Message: fmt.Sprintf("Failed to extract metric name for the ruleGroup %q and the alertingRule %q", ruleGroup.Name, v.Name),
						Error:   parserErr,
					})
					continue
				}
				populateUsage(metricUsage,
					metricNames,
					modelAPIV1.RuleUsage{
						PromLink:   source,
						GroupName:  ruleGroup.Name,
						Name:       v.Name,
						Expression: v.Query,
					},
					true,
				)
				populateUsage(partialMetricUsage,
					partialMetrics,
					modelAPIV1.RuleUsage{
						PromLink:   source,
						GroupName:  ruleGroup.Name,
						Name:       v.Name,
						Expression: v.Query,
					},
					true,
				)
			default:
				errs = append(errs, &modelAPIV1.LogError{
					Error: fmt.Errorf("unknown rule type %T", rule),
				})
			}
		}
	}
	return metricUsage, partialMetricUsage, errs
}

// AnalyzePromQLExpression is returning a list of valid metric names extracted from the PromQL expression.
// It also returned a list of partial metric names that likely look like a regexp.
// This function is kept for backward compatibility and delegates to the provided expression analyzer.
func AnalyzePromQLExpression(query string, analyzer expr.Analyzer) (set.Set[string], set.Set[string], error) {
	if analyzer == nil {
		return nil, nil, fmt.Errorf("expression analyzer is not configured")
	}
	return analyzer.Analyze(query)
}

func IsValidMetricName(name string) bool {
	return validMetricName.MatchString(name)
}

func populateUsage(metricUsage map[string]*modelAPIV1.MetricUsage, metricNames set.Set[string], item modelAPIV1.RuleUsage, isAlertingRules bool) {
	for metricName := range metricNames {
		if usage, ok := metricUsage[metricName]; ok {
			if isAlertingRules {
				usage.AlertRules.Add(item)
			} else {
				usage.RecordingRules.Add(item)
			}
		} else {
			u := &modelAPIV1.MetricUsage{}
			if isAlertingRules {
				u.AlertRules = set.New(item)
				u.RecordingRules = set.New[modelAPIV1.RuleUsage]()
			} else {
				u.RecordingRules = set.New(item)
				u.AlertRules = set.New[modelAPIV1.RuleUsage]()
			}
			metricUsage[metricName] = u
		}
	}
}
