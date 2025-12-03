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
	"fmt"

	modelAPIV1 "github.com/perses/metrics-usage/pkg/api/v1"
)

// Analyzer is the interface for parsing query expressions and extracting metric names
type Analyzer interface {
	Analyze(query string) (modelAPIV1.Set[string], modelAPIV1.Set[string], error)
}

var (
	currentAnalyzer Analyzer
)

const (
	EnginePromQL    = "promql"
	EngineMetricsQL = "metricsql"
)

// SetEngine sets the active analyzer engine
func SetEngine(engine string) error {
	switch engine {
	case EnginePromQL:
		currentAnalyzer = &promqlAnalyzer{}
	case EngineMetricsQL:
		currentAnalyzer = &metricsqlAnalyzer{}
	default:
		return fmt.Errorf("unknown engine %q, must be %q or %q", engine, EnginePromQL, EngineMetricsQL)
	}
	return nil
}

// Analyze parses the query expression and extracts metric names using the active analyzer
func Analyze(query string) (modelAPIV1.Set[string], modelAPIV1.Set[string], error) {
	analyzer := currentAnalyzer
	if analyzer == nil {
		// Default to promql if not set
		analyzer = &promqlAnalyzer{}
	}

	return analyzer.Analyze(query)
}
