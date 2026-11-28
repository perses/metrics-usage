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
	"github.com/prometheus/client_golang/prometheus"
)

type telemetry struct {
	db                  Database
	metricsAnalyzedDesc *prometheus.Desc
}

// NewTelemetry returns a Prometheus collector exposing the state of the database.
// The metrics are computed from the database each time they are collected.
func NewTelemetry(namespace string, db Database) prometheus.Collector {
	return &telemetry{
		db: db,
		metricsAnalyzedDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "metrics_analyzed"),
			"Number of metrics analyzed, partitioned by whether they are used by a dashboard or a rule.",
			[]string{"used"},
			nil,
		),
	}
}

func (t *telemetry) Describe(ch chan<- *prometheus.Desc) {
	ch <- t.metricsAnalyzedDesc
}

func (t *telemetry) Collect(ch chan<- prometheus.Metric) {
	used, unused := t.db.CountMetrics()
	ch <- prometheus.MustNewConstMetric(t.metricsAnalyzedDesc, prometheus.GaugeValue, float64(used), "true")
	ch <- prometheus.MustNewConstMetric(t.metricsAnalyzedDesc, prometheus.GaugeValue, float64(unused), "false")
}
