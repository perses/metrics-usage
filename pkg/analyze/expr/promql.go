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

package expr

import (
	"regexp"

	"github.com/perses/common/set"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/promql/parser"
)

var validMetricName = regexp.MustCompile(`^[a-zA-Z_:][a-zA-Z0-9_:]*$`)

type promqlAnalyzer struct{}

func (a *promqlAnalyzer) Analyze(query string) (set.Set[string], set.Set[string], error) {
	expr, err := parser.ParseExpr(query)
	if err != nil {
		return nil, nil, err
	}
	metricNames := set.Set[string]{}
	partialMetricNames := set.Set[string]{}
	parser.Inspect(expr, func(node parser.Node, _ []parser.Node) error {
		if n, ok := node.(*parser.VectorSelector); ok {
			// The metric name is only present when the node is a VectorSelector.
			// Then if the vector has the for metric_name{labelName="labelValue"}, then .Name is set.
			// Otherwise, we need to look at the labelName __name__ to find it.
			// Note: we will need to change this rule with Prometheus 3.0
			if n.Name != "" {
				metricNames.Add(n.Name)
				return nil
			}
			for _, m := range n.LabelMatchers {
				if m.Name == model.MetricNameLabel {
					if isValidMetricName(m.Value) {
						metricNames.Add(m.Value)
					} else {
						partialMetricNames.Add(m.Value)
					}

					return nil
				}
			}
		}
		return nil
	})
	return metricNames, partialMetricNames, nil
}

func isValidMetricName(name string) bool {
	return validMetricName.MatchString(name)
}
