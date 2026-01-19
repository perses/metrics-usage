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
	"github.com/VictoriaMetrics/metricsql"
	"github.com/perses/common/set"
	"github.com/prometheus/common/model"
)

type metricsqlAnalyzer struct{}

func (a *metricsqlAnalyzer) Analyze(query string) (set.Set[string], set.Set[string], error) {
	expr, err := metricsql.Parse(query)
	if err != nil {
		return nil, nil, err
	}

	metricNames := set.Set[string]{}
	partialMetricNames := set.Set[string]{}

	// Walk the AST to find metric selectors
	metricsql.VisitAll(expr, func(e metricsql.Expr) {
		if me, ok := e.(*metricsql.MetricExpr); ok {
			// LabelFilterss is a slice of OR-delimited groups of label filters
			// Each group is a slice of LabelFilter
			for _, filterGroup := range me.LabelFilterss {
				for _, filter := range filterGroup {
					if filter.Label == model.MetricNameLabel {
						value := filter.Value
						// If it's a regexp or not a valid metric name, treat as partial
						if filter.IsRegexp || !isValidMetricName(value) {
							partialMetricNames.Add(value)
						} else {
							metricNames.Add(value)
						}
					}
				}
			}
		}
	})

	return metricNames, partialMetricNames, nil
}
