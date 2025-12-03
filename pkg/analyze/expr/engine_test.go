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
	"testing"
)

func TestSetEngine(t *testing.T) {
	tests := []struct {
		name    string
		engine  string
		wantErr bool
	}{
		{
			name:    "valid promql engine",
			engine:  EnginePromQL,
			wantErr: false,
		},
		{
			name:    "valid metricsql engine",
			engine:  EngineMetricsQL,
			wantErr: false,
		},
		{
			name:    "invalid engine",
			engine:  "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := SetEngine(tt.engine)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetEngine() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAnalyze_PromQL(t *testing.T) {
	err := SetEngine(EnginePromQL)
	if err != nil {
		t.Fatalf("SetEngine() error = %v", err)
	}

	tests := []struct {
		name        string
		query       string
		wantMetrics []string
		wantPartial []string
		wantErr     bool
	}{
		{
			name:        "simple metric selector",
			query:       `http_requests_total{job="api"}`,
			wantMetrics: []string{"http_requests_total"},
			wantPartial: []string{},
			wantErr:     false,
		},
		{
			name:        "metric with function",
			query:       `sum(rate(foo_bar[5m])) by (job)`,
			wantMetrics: []string{"foo_bar"},
			wantPartial: []string{},
			wantErr:     false,
		},
		{
			name:        "metric via __name__ label",
			query:       `{__name__="baz_qux", instance="i"}`,
			wantMetrics: []string{"baz_qux"},
			wantPartial: []string{},
			wantErr:     false,
		},
		{
			name:        "regex metric name",
			query:       `{__name__=~"baz_.*"}`,
			wantMetrics: []string{},
			wantPartial: []string{"baz_.*"},
			wantErr:     false,
		},
		{
			name:        "invalid query",
			query:       `{invalid syntax}`,
			wantMetrics: nil,
			wantPartial: nil,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics, partial, err := Analyze(tt.query)
			if (err != nil) != tt.wantErr {
				t.Errorf("Analyze() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			// Check metrics
			if len(metrics) != len(tt.wantMetrics) {
				t.Errorf("Analyze() metrics length = %d, want %d", len(metrics), len(tt.wantMetrics))
			}
			for _, m := range tt.wantMetrics {
				if !metrics.Contains(m) {
					t.Errorf("Analyze() missing metric %q", m)
				}
			}

			// Check partial metrics
			if len(partial) != len(tt.wantPartial) {
				t.Errorf("Analyze() partial length = %d, want %d", len(partial), len(tt.wantPartial))
			}
			for _, m := range tt.wantPartial {
				if !partial.Contains(m) {
					t.Errorf("Analyze() missing partial metric %q", m)
				}
			}
		})
	}
}

func TestAnalyze_MetricsQL(t *testing.T) {
	err := SetEngine("metricsql")
	if err != nil {
		t.Fatalf("SetEngine() error = %v", err)
	}

	tests := []struct {
		name        string
		query       string
		wantMetrics []string
		wantPartial []string
		wantErr     bool
	}{
		{
			name:        "simple metric selector",
			query:       `http_requests_total{job="api"}`,
			wantMetrics: []string{"http_requests_total"},
			wantPartial: []string{},
			wantErr:     false,
		},
		{
			name:        "metric with function",
			query:       `sum(rate(foo_bar[5m])) by (job)`,
			wantMetrics: []string{"foo_bar"},
			wantPartial: []string{},
			wantErr:     false,
		},
		{
			name:        "metric via __name__ label",
			query:       `{__name__="baz_qux", instance="i"}`,
			wantMetrics: []string{"baz_qux"},
			wantPartial: []string{},
			wantErr:     false,
		},
		{
			name:        "regex metric name",
			query:       `{__name__=~"baz_.*"}`,
			wantMetrics: []string{},
			wantPartial: []string{"baz_.*"},
			wantErr:     false,
		},
		{
			name:        "invalid query",
			query:       `{invalid syntax}`,
			wantMetrics: nil,
			wantPartial: nil,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics, partial, err := Analyze(tt.query)
			if (err != nil) != tt.wantErr {
				t.Errorf("Analyze() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			// Check metrics
			if len(metrics) != len(tt.wantMetrics) {
				t.Errorf("Analyze() metrics length = %d, want %d", len(metrics), len(tt.wantMetrics))
			}
			for _, m := range tt.wantMetrics {
				if !metrics.Contains(m) {
					t.Errorf("Analyze() missing metric %q", m)
				}
			}

			// Check partial metrics
			if len(partial) != len(tt.wantPartial) {
				t.Errorf("Analyze() partial length = %d, want %d", len(partial), len(tt.wantPartial))
			}
			for _, m := range tt.wantPartial {
				if !partial.Contains(m) {
					t.Errorf("Analyze() missing partial metric %q", m)
				}
			}
		})
	}
}

func TestAnalyze_DefaultEngine(t *testing.T) {
	// Reset to nil to test default behavior
	currentAnalyzer = nil

	metrics, _, err := Analyze(`http_requests_total{job="api"}`)
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	// Should default to promql and work
	if !metrics.Contains("http_requests_total") {
		t.Errorf("Analyze() missing metric with default engine")
	}

	// Set back to promql for other tests
	_ = SetEngine(EnginePromQL)
}
