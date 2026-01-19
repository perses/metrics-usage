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
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/brunoga/deep"
	"github.com/perses/common/set"
	"github.com/perses/metrics-usage/config"
	v1 "github.com/perses/metrics-usage/pkg/api/v1"
	"github.com/perses/perses/pkg/model/api/v1/common"
	"github.com/sirupsen/logrus"
)

var replaceVariableRegexp = regexp.MustCompile(`\$\{[a-zA-Z0-9_:]+}`)

type Database interface {
	DeleteMetric(name string) bool
	GetMetric(name string) *v1.Metric
	ListMetrics() (map[string]*v1.Metric, error)
	ListPartialMetrics() (map[string]*v1.PartialMetric, error)
	ListPendingUsage() map[string]*v1.MetricUsage
	EnqueueMetricList(metrics []string)
	EnqueuePartialMetricsUsage(usages map[string]*v1.MetricUsage)
	EnqueueUsage(usages map[string]*v1.MetricUsage)
	EnqueueLabels(labels map[string][]string)
}

func New(cfg config.Database) Database {
	d := &db{
		metrics:                  make(map[string]*v1.Metric),
		metricLastTimeSeen:       make(map[string]uint8),
		threshold:                cfg.Threshold,
		partialMetrics:           make(map[string]*v1.PartialMetric),
		usage:                    make(map[string]*v1.MetricUsage),
		usageQueue:               make(chan map[string]*v1.MetricUsage, 250),
		partialMetricsUsageQueue: make(chan map[string]*v1.MetricUsage, 250),
		labelsQueue:              make(chan map[string][]string, 250),
		metricsQueue:             make(chan []string, 10),
		path:                     cfg.Path,
	}

	go d.watchUsageQueue()
	go d.watchMetricsQueue()
	go d.watchPartialMetricsUsageQueue()
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
	// metrics is the list of metric name (as a key) associated with their usage based on the different collector activated.
	// This struct is our "database".
	metrics map[string]*v1.Metric
	// metricLastTimeSeen is the list of the metric associated with a counter.
	// The counter represents the number of times we didn't see the metric.
	// Once a counter reaches a certain threshold, we will remove the metric from the database.
	// Note: the size of this map is the same as the size of metrics.
	metricLastTimeSeen map[string]uint8
	// threshold is the number of times we didn't see a metric before removing it from the database.
	threshold uint8
	// partialMetrics is the list of metric name that likely contains a variable or a regexp and as such cannot be a valid metric name.
	partialMetrics map[string]*v1.PartialMetric
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
	// partialMetricsUsageQueue is the way to send the usage per metric that is not valid to write in the database.
	// There will be no other way to write in it.
	// Doing that allows us to accept more HTTP requests to write data and to delay the actual writing.
	partialMetricsUsageQueue chan map[string]*v1.MetricUsage
	// path is the path to the JSON file where metrics is flushed periodically
	// It is empty if the database is purely in memory.
	path string
	// We are expecting to spend more time to write data than actually read.
	// Which results in having too many writers
	// and so unable to read the data because the lock queue is too long to be able to access to the data.
	// If this scenario happens,
	// 1. Then let's flush the data into a file periodically (or once the queue is empty (if it happens))
	// 2. Read the file directly when a read query is coming
	// Like that we have two different ways to read and write the data.
	metricsMutex             sync.Mutex
	partialMetricsUsageMutex sync.Mutex
}

var _ = Database(&db{})

func (d *db) DeleteMetric(name string) bool {
	d.metricsMutex.Lock()
	defer d.metricsMutex.Unlock()
	if _, ok := d.metrics[name]; !ok {
		return false
	}
	d.deleteMetric(name)
	return true
}

func (d *db) GetMetric(name string) *v1.Metric {
	d.metricsMutex.Lock()
	defer d.metricsMutex.Unlock()
	return d.metrics[name]
}

func (d *db) ListMetrics() (map[string]*v1.Metric, error) {
	d.metricsMutex.Lock()
	defer d.metricsMutex.Unlock()
	return deep.Copy(d.metrics)
}

func (d *db) ListPartialMetrics() (map[string]*v1.PartialMetric, error) {
	d.partialMetricsUsageMutex.Lock()
	defer d.partialMetricsUsageMutex.Unlock()
	return deep.Copy(d.partialMetrics)
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

func (d *db) EnqueuePartialMetricsUsage(usages map[string]*v1.MetricUsage) {
	d.partialMetricsUsageQueue <- usages
}

func (d *db) EnqueueLabels(labels map[string][]string) {
	d.labelsQueue <- labels
}

// watchMetricsQueue is the way to store the metric name in the database.
func (d *db) watchMetricsQueue() {
	for metricsName := range d.metricsQueue {
		d.metricsMutex.Lock()
		// We are not simply storing the metric here.
		// While we are doing that, we are also checking if during the time to collect the metric we didn't already receive usage of the metrics.
		// If it's the case, then we can already associate a usage with the metric.
		// We are also re-setting the counter that tells us when it is the last time we saw the metric.
		//
		// First, let's increase the counter for every previous metric stored.
		for metricName := range d.metricLastTimeSeen {
			d.metricLastTimeSeen[metricName]++
		}

		// Then, we are looping other the metrics received and check if we already have them in the database.
		for _, metricName := range metricsName {
			if _, ok := d.metrics[metricName]; !ok {
				// As this queue only serves the purpose of storing missing metrics, we are only looking for the one not already present in the database.
				d.metrics[metricName] = &v1.Metric{
					Labels: make(set.Set[string]),
				}
				// This will be used to associate a metric with a partial one.
				d.matchValidMetric(metricName)
				// Since it's a new metric, potentially we already have a usage stored in the buffer.
				if usage, usageExists := d.usage[metricName]; usageExists {
					// TODO at some point we need to erase the usage map because it will cause a memory leak
					d.metrics[metricName].Usage = usage
					delete(d.usage, metricName)
				}
			}
			// In any case, whether the metric is already known or not, we are re-setting the counter.
			d.metricLastTimeSeen[metricName] = 0
		}
		// Finally, we are looping other all counters and remove metrics not seen since a while.
		for metricName, counter := range d.metricLastTimeSeen {
			if counter < d.threshold {
				continue
			}
			d.deleteMetric(metricName)
		}
		d.metricsMutex.Unlock()
	}
}

func (d *db) watchPartialMetricsUsageQueue() {
	for data := range d.partialMetricsUsageQueue {
		d.partialMetricsUsageMutex.Lock()
		for metricName, usage := range data {
			if _, ok := d.partialMetrics[metricName]; !ok {
				re, matchingMetrics := d.matchPartialMetric(metricName)
				d.partialMetrics[metricName] = &v1.PartialMetric{
					Usage:           usage,
					MatchingMetrics: matchingMetrics,
					MatchingRegexp:  re,
				}
			} else {
				d.partialMetrics[metricName].Usage = v1.MergeUsage(d.partialMetrics[metricName].Usage, usage)
			}
		}
		d.partialMetricsUsageMutex.Unlock()
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
				d.usage[metricName] = v1.MergeUsage(d.usage[metricName], usage)
			} else {
				d.metrics[metricName].Usage = v1.MergeUsage(d.metrics[metricName].Usage, usage)
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
					Labels: set.New(labels...),
				}
			} else {
				if d.metrics[metricName].Labels == nil {
					d.metrics[metricName].Labels = set.New(labels...)
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

func (d *db) deleteMetric(name string) {
	// This method does not lock the mutex associated with the metrics
	// because it is supposed to be called from a parent function that already did it.
	delete(d.metrics, name)
	delete(d.metricLastTimeSeen, name)
	d.partialMetricsUsageMutex.Lock()
	defer d.partialMetricsUsageMutex.Unlock()
	for _, partialMetric := range d.partialMetrics {
		if partialMetric.MatchingMetrics.Contains(name) {
			partialMetric.MatchingMetrics.Remove(name)
		}
	}
}

func (d *db) matchPartialMetric(partialMetric string) (*common.Regexp, set.Set[string]) {
	re, err := generateRegexp(partialMetric)
	if err != nil {
		logrus.WithError(err).Errorf("unable to compile the partial metric name %q into a regexp", partialMetric)
		return nil, nil
	}
	if re == nil {
		return nil, nil
	}
	result := set.New[string]()
	d.metricsMutex.Lock()
	defer d.metricsMutex.Unlock()
	for m := range d.metrics {
		if re.MatchString(m) {
			result.Add(m)
		}
	}
	return re, result
}

func (d *db) matchValidMetric(validMetric string) {
	d.partialMetricsUsageMutex.Lock()
	defer d.partialMetricsUsageMutex.Unlock()
	for metricName, partialMetric := range d.partialMetrics {
		re := partialMetric.MatchingRegexp
		if re == nil {
			var err error
			re, err = generateRegexp(metricName)
			if err != nil {
				logrus.WithError(err).Errorf("unable to compile the partial metric name %q into a regexp", metricName)
				continue
			}
			partialMetric.MatchingRegexp = re
			if re == nil {
				continue
			}
		}
		if isMatching(re, validMetric) {
			matchingMetrics := partialMetric.MatchingMetrics
			if matchingMetrics == nil {
				matchingMetrics = set.New[string]()
				partialMetric.MatchingMetrics = matchingMetrics
			}
			matchingMetrics.Add(validMetric)
		}
	}
}

// GenerateRegexp is taking a partial metric name,
// will replace every variable by a pattern and then returning a regepx if the final string is not just equal to .*.
func generateRegexp(partialMetricName string) (*common.Regexp, error) {
	// The first step is to replace every variable by a single special char.
	// We are using a special single char because it will be easier to find if these chars are continuous
	// or if there are other characters in between.
	s := replaceVariableRegexp.ReplaceAllString(partialMetricName, "#")
	s = strings.ReplaceAll(s, ".+", "#")
	s = strings.ReplaceAll(s, ".*", "#")
	if s == "#" || len(s) == 0 {
		// This means the metric name is just a variable and as such can match all metric.
		// So it's basically impossible to know what this partial metric name is covering/matching.
		return nil, nil
	}
	// The next step is to contact every continuous special char '#' to a single one.
	compileString := fmt.Sprintf("%c", s[0])
	expr := []rune(s)
	for i := 1; i < len(expr); i++ {
		if expr[i-1] == '#' && expr[i-1] == expr[i] {
			continue
		}
		compileString += string(expr[i])
	}
	if compileString == "#" {
		return nil, nil
	}
	compileString = strings.ReplaceAll(compileString, "#", ".+")
	re, err := common.NewRegexp(fmt.Sprintf("^%s$", compileString))
	return &re, err
}

func isMatching(re *common.Regexp, metric string) bool {
	if !re.MatchString(metric) {
		return false
	}
	// We are taking some time to look at what is matching.
	// We are doing that to avoid the situation where this regexp `foo|` is matching everything.
	for _, match := range re.FindAllString(metric, -1) {
		if len(match) > 0 {
			return true
		}
	}
	return false
}
