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

package database

import (
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/perses/metrics-usage/config"
	v1 "github.com/perses/metrics-usage/pkg/api/v1"
	"github.com/sirupsen/logrus"
)

type Database interface {
	GetMetric(name string) *v1.Metric
	ListMetrics() map[string]*v1.Metric
	ListInvalidMetrics() map[string]*v1.Metric
	ListPendingUsage() map[string]*v1.MetricUsage
	EnqueueMetricList(metrics []string)
	EnqueueInvalidMetricsUsage(usages map[string]*v1.MetricUsage)
	EnqueueUsage(usages map[string]*v1.MetricUsage)
	EnqueueLabels(labels map[string][]string)
}

func New(cfg config.Database) Database {
	d := &db{
		metrics:                  make(map[string]*v1.Metric),
		invalidMetrics:           make(map[string]*v1.Metric),
		usage:                    make(map[string]*v1.MetricUsage),
		usageQueue:               make(chan map[string]*v1.MetricUsage, 250),
		invalidMetricsUsageQueue: make(chan map[string]*v1.MetricUsage, 250),
		labelsQueue:              make(chan map[string][]string, 250),
		metricsQueue:             make(chan []string, 10),
		path:                     cfg.Path,
	}

	go d.watchUsageQueue()
	go d.watchMetricsQueue()
	go d.watchInvalidMetricsUsageQueue()
	go d.watchLabelsQueue()
	if !*cfg.InMemory {
		if err := d.readMetricsInJSONFile(); err != nil {
			logrus.WithError(err).Warning("failed to read metrics file")
		}
		go d.flush(time.Duration(cfg.FlushPeriod))
	}
	return d
}

type db struct {
	Database
	// metrics is the list of metric name (as a key) associated to their usage based on the different collector activated.
	// This struct is our "database".
	metrics map[string]*v1.Metric
	// invalidMetrics is the list of metric name that likely contains a variable or a regexp and as such cannot be a valid metric name.
	invalidMetrics map[string]*v1.Metric
	// usage is a buffer in case the metric name has not yet been collected
	usage map[string]*v1.MetricUsage
	// metricsQueue is the channel that should be used to send and receive the list of metric name to keep in memory.
	// Based on this list, we will then collect their usage.
	metricsQueue chan []string
	// labelsQueue is the way to send the labels per metric to write in the database.
	// There will be no other way to write in it.
	// Doing that allows us to accept more HTTP requests to write data and to delay the actual writing.
	labelsQueue chan map[string][]string
	// usageQueue is the way to send the usage per metric to write in the database.
	// There will be no other way to write in it.
	// Doing that allows us to accept more HTTP requests to write data and to delay the actual writing.
	usageQueue chan map[string]*v1.MetricUsage
	// invalidMetricsUsageQueue is the way to send the usage per metric that is not valid to write in the database.
	// There will be no other way to write in it.
	// Doing that allows us to accept more HTTP requests to write data and to delay the actual writing.
	invalidMetricsUsageQueue chan map[string]*v1.MetricUsage
	// path is the path to the JSON file where metrics is flushed periodically
	// It is empty if the database is purely in memory.
	path string
	// We are expecting to spend more time to write data than actually read.
	// Which result having too many writers,
	// and so unable to read the data because the lock queue is too long to be able to access to the data.
	// If this scenario happens,
	// 1. Then let's flush the data into a file periodically (or once the queue is empty (if it happens))
	// 2. Read the file directly when a read query is coming
	// Like that we have two different ways to read and write the data.
	metricsMutex             sync.Mutex
	invalidMetricsUsageMutex sync.Mutex
}

func (d *db) GetMetric(name string) *v1.Metric {
	d.metricsMutex.Lock()
	defer d.metricsMutex.Unlock()
	return d.metrics[name]
}

func (d *db) ListMetrics() map[string]*v1.Metric {
	d.metricsMutex.Lock()
	defer d.metricsMutex.Unlock()
	return d.metrics
}

func (d *db) ListInvalidMetrics() map[string]*v1.Metric {
	d.invalidMetricsUsageMutex.Lock()
	defer d.invalidMetricsUsageMutex.Unlock()
	return d.invalidMetrics
}

func (d *db) EnqueueMetricList(metrics []string) {
	d.metricsQueue <- metrics
}

func (d *db) ListPendingUsage() map[string]*v1.MetricUsage {
	d.metricsMutex.Lock()
	defer d.metricsMutex.Unlock()
	return d.usage
}

func (d *db) EnqueueUsage(usages map[string]*v1.MetricUsage) {
	d.usageQueue <- usages
}

func (d *db) EnqueueInvalidMetricsUsage(usages map[string]*v1.MetricUsage) {
	d.invalidMetricsUsageQueue <- usages
}

func (d *db) EnqueueLabels(labels map[string][]string) {
	d.labelsQueue <- labels
}

func (d *db) watchMetricsQueue() {
	for metricsName := range d.metricsQueue {
		d.metricsMutex.Lock()
		for _, metricName := range metricsName {
			if _, ok := d.metrics[metricName]; !ok {
				// As this queue only serves the purpose of storing missing metrics, we are only looking for the one not already present in the database.
				d.metrics[metricName] = &v1.Metric{
					Labels: make(v1.Set[string]),
				}
				// Since it's a new metric, potentially we already have a usage stored in the buffer.
				if usage, usageExists := d.usage[metricName]; usageExists {
					// TODO at some point we need to erase the usage map because it will cause a memory leak
					d.metrics[metricName].Usage = usage
					delete(d.usage, metricName)
				}
			}
		}
		d.metricsMutex.Unlock()
	}
}

func (d *db) watchInvalidMetricsUsageQueue() {
	for data := range d.invalidMetricsUsageQueue {
		d.invalidMetricsUsageMutex.Lock()
		for metricName, usage := range data {
			if _, ok := d.invalidMetrics[metricName]; !ok {
				d.invalidMetrics[metricName] = &v1.Metric{
					Usage: usage,
				}
			} else {
				d.invalidMetrics[metricName].Usage = mergeUsage(d.invalidMetrics[metricName].Usage, usage)
			}
		}
		d.invalidMetricsUsageMutex.Unlock()
	}
}

func (d *db) watchUsageQueue() {
	for data := range d.usageQueue {
		d.metricsMutex.Lock()
		for metricName, usage := range data {
			if _, ok := d.metrics[metricName]; !ok {
				logrus.Debugf("metric_name %q is used but it's not found by the metric collector", metricName)
				// Since the metric_name is not known yet, we need to buffer it.
				// In a later stage, if the metric is received/known,
				// we will then use this buffer to populate the usage of the metric.
				if _, isUsed := d.usage[metricName]; !isUsed {
					d.usage[metricName] = usage
				} else {
					// In that case, we need to merge the usage.
					d.usage[metricName] = mergeUsage(d.usage[metricName], usage)
				}
			} else {
				d.metrics[metricName].Usage = mergeUsage(d.metrics[metricName].Usage, usage)
			}
		}
		d.metricsMutex.Unlock()
	}
}

func (d *db) watchLabelsQueue() {
	for data := range d.labelsQueue {
		d.metricsMutex.Lock()
		for metricName, labels := range data {
			if _, ok := d.metrics[metricName]; !ok {
				// In this case, we should add the metric, because it means the metrics has been found from another source.
				d.metrics[metricName] = &v1.Metric{
					Labels: v1.NewSet(labels...),
				}
			} else {
				if d.metrics[metricName].Labels == nil {
					d.metrics[metricName].Labels = v1.NewSet(labels...)
				} else {
					d.metrics[metricName].Labels.Add(labels...)
				}
			}
		}
		d.metricsMutex.Unlock()
	}
}

func (d *db) flush(period time.Duration) {
	ticker := time.NewTicker(period)
	defer ticker.Stop()
	for range ticker.C {
		if err := d.writeMetricsInJSONFile(); err != nil {
			logrus.WithError(err).Error("unable to flush the data in the file")
		}
	}
}

func (d *db) writeMetricsInJSONFile() error {
	d.metricsMutex.Lock()
	defer d.metricsMutex.Unlock()
	data, err := json.Marshal(d.metrics)
	if err != nil {
		return err
	}
	return os.WriteFile(d.path, data, 0644)
}

func (d *db) readMetricsInJSONFile() error {
	data, err := os.ReadFile(d.path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &d.metrics)
}

func mergeUsage(old, new *v1.MetricUsage) *v1.MetricUsage {
	if old == nil {
		return new
	}
	if new == nil {
		return old
	}
	old.Dashboards.Merge(new.Dashboards)
	old.AlertRules.Merge(new.AlertRules)
	old.RecordingRules.Merge(new.RecordingRules)
	return old
}
