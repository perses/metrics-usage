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
	"encoding/json"
	"os"
	"slices"
	"testing"

	"github.com/perses/common/set"
	"github.com/perses/metrics-usage/pkg/analyze/expr"
	modelAPIV1 "github.com/perses/metrics-usage/pkg/api/v1"
	"github.com/stretchr/testify/assert"
)

func unmarshalDashboard(path string) (*SimplifiedDashboard, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	result := &SimplifiedDashboard{}
	return result, json.Unmarshal(data, result)
}

func TestAnalyze(t *testing.T) {
	analyzer, err := expr.NewAnalyzer(expr.EnginePromQL)
	if err != nil {
		t.Fatalf("failed to initialize analyzer: %v", err)
	}
	tests := []struct {
		name           string
		dashboardFile  string
		resultMetrics  []string
		invalidMetrics []string
		resultErrs     []*modelAPIV1.LogError
	}{
		{
			name:          "from/to variables",
			dashboardFile: "tests/d1.json",
			resultMetrics: []string{"run", "service_color"},
		},
		{
			name:          "use static variable",
			dashboardFile: "tests/d2.json",
			resultMetrics: []string{"gacms_svc_elapsed_time_seconds_bucket"},
		},
		{
			name:          "variable replace order",
			dashboardFile: "tests/d3.json",
			resultMetrics: []string{"probe_success"},
		},
		{
			name:          "variable in metrics",
			dashboardFile: "tests/d4.json",
			resultMetrics: []string{
				"otelcol_exporter_queue_capacity",
				"otelcol_exporter_queue_size",
				"otelcol_process_memory_rss",
				"otelcol_process_runtime_heap_alloc_bytes",
				"otelcol_process_runtime_total_sys_memory_bytes",
				"otelcol_processor_batch_batch_send_size_bucket",
				"otelcol_processor_batch_batch_send_size_count",
				"otelcol_processor_batch_batch_send_size_sum",
				"otelcol_rpc_client_duration_bucket",
				"otelcol_rpc_client_request_size_bucket",
				"otelcol_rpc_client_responses_per_rpc_count",
				"otelcol_rpc_server_duration_bucket",
				"otelcol_rpc_server_request_size_bucket",
				"otelcol_rpc_server_responses_per_rpc_count",
			},
			invalidMetrics: []string{
				"otelcol_exporter_.+",
				"otelcol_exporter_enqueue_failed_log_records${suffix}",
				"otelcol_exporter_enqueue_failed_metric_points${suffix}",
				"otelcol_exporter_enqueue_failed_spans${suffix}",
				"otelcol_exporter_send_failed_log_records${suffix}",
				"otelcol_exporter_send_failed_metric_points${suffix}",
				"otelcol_exporter_send_failed_spans${suffix}",
				"otelcol_exporter_sent_log_records${suffix}",
				"otelcol_exporter_sent_metric_points${suffix}",
				"otelcol_exporter_sent_spans${suffix}",
				"otelcol_otelsvc_k8s_namespace_added${suffix}",
				"otelcol_otelsvc_k8s_namespace_updated${suffix}",
				"otelcol_otelsvc_k8s_pod_added${suffix}",
				"otelcol_otelsvc_k8s_pod_deleted${suffix}",
				"otelcol_otelsvc_k8s_pod_updated${suffix}",
				"otelcol_process_cpu_seconds${suffix}",
				"otelcol_process_uptime${suffix}",
				"otelcol_process_uptime.+",
				"otelcol_processor_.+",
				"otelcol_processor_accepted_log_records${suffix}",
				"otelcol_processor_accepted_metric_points${suffix}",
				"otelcol_processor_accepted_spans${suffix}",
				"otelcol_processor_batch_batch_size_trigger_send${suffix}",
				"otelcol_processor_batch_timeout_trigger_send${suffix}",
				"otelcol_processor_dropped_log_records${suffix}",
				"otelcol_processor_dropped_metric_points${suffix}",
				"otelcol_processor_dropped_spans${suffix}",
				"otelcol_processor_refused_log_records${suffix}",
				"otelcol_processor_refused_metric_points${suffix}",
				"otelcol_processor_refused_spans${suffix}",
				"otelcol_receiver_.+",
				"otelcol_receiver_accepted_log_records${suffix}",
				"otelcol_receiver_accepted_metric_points${suffix}",
				"otelcol_receiver_accepted_spans${suffix}",
				"otelcol_receiver_refused_log_records${suffix}",
				"otelcol_receiver_refused_metric_points${suffix}",
				"otelcol_receiver_refused_spans${suffix}",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dashboard, err := unmarshalDashboard(tt.dashboardFile)
			if err != nil {
				t.Fatal(err)
			}
			metrics, partialMetrics, errs := Analyze(dashboard, analyzer)
			metricsAsSlice := metrics.TransformAsSlice()
			invalidMetricsAsSlice := partialMetrics.TransformAsSlice()
			slices.Sort(metricsAsSlice)
			slices.Sort(invalidMetricsAsSlice)
			assert.Equal(t, tt.resultMetrics, metricsAsSlice)
			assert.Equal(t, tt.invalidMetrics, invalidMetricsAsSlice)
			assert.Equal(t, tt.resultErrs, errs)
		})
	}
}

func TestAnalyzeWithFilter(t *testing.T) {
	analyzer, err := expr.NewAnalyzer(expr.EnginePromQL)
	if err != nil {
		t.Fatalf("failed to initialize analyzer: %v", err)
	}

	tests := []struct {
		name           string
		dashboardFile  string
		filter         *DatasourceFilter
		resultMetrics  []string
		invalidMetrics []string
	}{
		{
			name:           "no filter - all metrics extracted",
			dashboardFile:  "tests/d5.json",
			filter:         nil,
			resultMetrics:  []string{"cpu_usage", "memory_usage", "up"},
			invalidMetrics: []string{},
		},
		{
			name:          "filter by postgres type - postgres target ignored",
			dashboardFile: "tests/d5.json",
			filter: &DatasourceFilter{
				IgnoreTypes: set.New("postgres"),
			},
			resultMetrics:  []string{"cpu_usage", "memory_usage", "up"},
			invalidMetrics: []string{},
		},
		{
			name:          "filter by multiple types - postgres and mysql ignored",
			dashboardFile: "tests/d5.json",
			filter: &DatasourceFilter{
				IgnoreTypes: set.New("postgres", "mysql"),
			},
			resultMetrics:  []string{"cpu_usage", "memory_usage", "up"},
			invalidMetrics: []string{},
		},
		{
			name:          "filter by UID - specific prometheus target ignored",
			dashboardFile: "tests/d5.json",
			filter: &DatasourceFilter{
				IgnoreUIDs: set.New("ignore-this-uid"),
			},
			resultMetrics:  []string{"memory_usage", "up"},
			invalidMetrics: []string{},
		},
		{
			name:          "filter by both type and UID",
			dashboardFile: "tests/d5.json",
			filter: &DatasourceFilter{
				IgnoreTypes: set.New("postgres"),
				IgnoreUIDs:  set.New("ignore-this-uid"),
			},
			resultMetrics:  []string{"memory_usage", "up"},
			invalidMetrics: []string{},
		},
		{
			name:          "filter case insensitive - postgres matches POSTGRES in target",
			dashboardFile: "tests/d6.json",
			filter: &DatasourceFilter{
				IgnoreTypes: set.New("postgres"),
			},
			resultMetrics:  []string{"up"},
			invalidMetrics: []string{},
		},
		{
			name:           "string datasource format - no filter",
			dashboardFile:  "tests/d7.json",
			filter:         nil,
			resultMetrics:  []string{"cpu_usage", "memory_usage", "up"},
			invalidMetrics: []string{},
		},
		{
			name:          "string datasource format - filter by UID",
			dashboardFile: "tests/d7.json",
			filter: &DatasourceFilter{
				IgnoreUIDs: set.New("ignore-this-uid"),
			},
			resultMetrics:  []string{"cpu_usage", "up"},
			invalidMetrics: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dashboard, err := unmarshalDashboard(tt.dashboardFile)
			if err != nil {
				t.Fatalf("failed to unmarshal dashboard: %v", err)
			}

			metrics, partialMetrics, _ := AnalyzeWithFilter(dashboard, analyzer, tt.filter)

			metricsAsSlice := metrics.TransformAsSlice()
			invalidMetricsAsSlice := partialMetrics.TransformAsSlice()

			assert.Equal(t, tt.resultMetrics, metricsAsSlice)
			if len(tt.invalidMetrics) == 0 {
				assert.Empty(t, invalidMetricsAsSlice)
			} else {
				assert.Equal(t, tt.invalidMetrics, invalidMetricsAsSlice)
			}
		})
	}
}
