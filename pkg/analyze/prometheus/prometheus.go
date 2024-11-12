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

	modelAPIV1 "github.com/perses/metrics-usage/pkg/api/v1"
	"github.com/perses/metrics-usage/utils"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/promql/parser"
)

func Analyze(ruleGroups []v1.RuleGroup, source string) (map[string]*modelAPIV1.MetricUsage, []*modelAPIV1.LogError) {
	var errs []*modelAPIV1.LogError
	metricUsage := make(map[string]*modelAPIV1.MetricUsage)
	for _, ruleGroup := range ruleGroups {
		for _, rule := range ruleGroup.Rules {
			switch v := rule.(type) {
			case v1.RecordingRule:
				metricNames, parserErr := AnalyzePromQLExpression(v.Query)
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
			case v1.AlertingRule:
				metricNames, parserErr := AnalyzePromQLExpression(v.Query)
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
			default:
				errs = append(errs, &modelAPIV1.LogError{
					Error: fmt.Errorf("unknown rule type %T", rule),
				})
			}
		}
	}
	return metricUsage, errs
}

func AnalyzePromQLExpression(query string) ([]string, error) {
	expr, err := parser.ParseExpr(query)
	if err != nil {
		return nil, err
	}
	var result []string
	parser.Inspect(expr, func(node parser.Node, _ []parser.Node) error {
		if n, ok := node.(*parser.VectorSelector); ok {
			// The metric name is only present when the node is a VectorSelector.
			// Then if the vector has the for metric_name{labelName="labelValue"}, then .Name is set.
			// Otherwise, we need to look at the labelName __name__ to find it.
			// Note: we will need to change this rule with Prometheus 3.0
			if n.Name != "" {
				result = append(result, n.Name)
				return nil
			}
			for _, m := range n.LabelMatchers {
				if m.Name == labels.MetricName {
					result = append(result, m.Value)
					return nil
				}
			}
		}
		return nil
	})
	return result, nil
}

func populateUsage(metricUsage map[string]*modelAPIV1.MetricUsage, metricNames []string, item modelAPIV1.RuleUsage, isAlertingRules bool) {
	for _, metricName := range metricNames {
		if usage, ok := metricUsage[metricName]; ok {
			if isAlertingRules {
				usage.AlertRules = utils.InsertIfNotPresent(usage.AlertRules, item)
			} else {
				usage.RecordingRules = utils.InsertIfNotPresent(usage.RecordingRules, item)
			}
		} else {
			u := &modelAPIV1.MetricUsage{}
			if isAlertingRules {
				u.AlertRules = []modelAPIV1.RuleUsage{item}
			} else {
				u.RecordingRules = []modelAPIV1.RuleUsage{item}
			}
			metricUsage[metricName] = u
		}
	}
}