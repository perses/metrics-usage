// Copyright The Perses Authors
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

package database

import (
	"strings"
	"testing"

	"github.com/perses/common/set"
	v1 "github.com/perses/metrics-usage/pkg/api/v1"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func TestCountMetrics(t *testing.T) {
	usage := &v1.MetricUsage{
		Dashboards: set.New(v1.DashboardUsage{ID: "perses/nodeexporterfull", Name: "nodeexporterfull"}),
	}
	tests := []struct {
		title          string
		metrics        map[string]*v1.Metric
		partialMetrics map[string]*v1.PartialMetric
		used           int
		unused         int
	}{
		{
			title: "empty database",
		},
		{
			title: "metrics with and without usage",
			metrics: map[string]*v1.Metric{
				"node_cpu_seconds_total":           {Usage: usage},
				"node_memory_MemFree_bytes":        {Labels: set.New("instance")},
				"node_disk_read_bytes_total":       {},
				"node_network_receive_bytes_total": {Usage: usage},
			},
			used:   2,
			unused: 2,
		},
		{
			title: "metrics used through a partial metric",
			metrics: map[string]*v1.Metric{
				"node_cpu_seconds_total":               {Usage: usage},
				"node_disk_discard_time_seconds_total": {},
				"node_disk_read_bytes_total":           {},
				"node_memory_MemFree_bytes":            {},
			},
			partialMetrics: map[string]*v1.PartialMetric{
				"node_disk_.+": {
					Usage:           usage,
					MatchingMetrics: set.New("node_disk_discard_time_seconds_total", "node_disk_read_bytes_total"),
				},
				"node_cpu_.+": {
					Usage:           usage,
					MatchingMetrics: set.New("node_cpu_seconds_total"),
				},
				"node_memory_.+": {
					MatchingMetrics: set.New("node_memory_MemFree_bytes"),
				},
				"${metric}": {
					Usage: usage,
				},
			},
			used:   3,
			unused: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.title, func(t *testing.T) {
			d := &db{metrics: test.metrics, partialMetrics: test.partialMetrics}
			used, unused := d.CountMetrics()
			assert.Equal(t, test.used, used)
			assert.Equal(t, test.unused, unused)
		})
	}
}

func TestTelemetry(t *testing.T) {
	d := &db{
		metrics: map[string]*v1.Metric{
			"node_cpu_seconds_total": {
				Usage: &v1.MetricUsage{
					AlertRules: set.New(v1.RuleUsage{Name: "NodeCPUHighUsage"}),
				},
			},
			"node_memory_MemFree_bytes":  {},
			"node_disk_read_bytes_total": {},
		},
		partialMetrics: make(map[string]*v1.PartialMetric),
	}
	expected := `
# HELP metrics_usage_metrics_analyzed Number of metrics analyzed, partitioned by whether they are used by a dashboard or a rule.
# TYPE metrics_usage_metrics_analyzed gauge
metrics_usage_metrics_analyzed{used="false"} 2
metrics_usage_metrics_analyzed{used="true"} 1
`
	assert.NoError(t, testutil.CollectAndCompare(NewTelemetry("metrics_usage", d), strings.NewReader(expected)))
}
