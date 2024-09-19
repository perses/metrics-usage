package database

import (
	"sync"

	v1 "github.com/perses/metrics-usage/pkg/api/v1"
	"github.com/sirupsen/logrus"
)

type Database interface {
	GetMetric(name string) *v1.Metric
	ListMetrics() map[string]*v1.Metric
	EnqueueMetricList(metrics []string)
	EnqueueUsage(usages map[string]*v1.MetricUsage)
}

func New() Database {
	d := &db{
		Database:     nil,
		metrics:      make(map[string]*v1.Metric),
		usageQueue:   make(chan map[string]*v1.MetricUsage, 250),
		metricsQueue: make(chan []string, 10),
	}

	go d.watchUsageQueue()
	go d.watchMetricsQueue()
	return d
}

type db struct {
	Database
	// metrics is the list of metric name (as a key) associated to their usage based on the different collector activated.
	// This struct is our "database".
	metrics map[string]*v1.Metric
	// metricsQueue is the channel that should be used to send and receive the list of metric name to keep in memory.
	// Based on this list, we will then collect their usage.
	metricsQueue chan []string
	// usageQueue is the way to send the usage per metric to write in the database.
	// There will be no other way to write in it.
	// Doing that allows us to accept more HTTP requests to write data and to delay the actual writing.
	usageQueue chan map[string]*v1.MetricUsage
	// We are expecting to spend more time to write data than actually read.
	// Which result having too many writers,
	// and so unable to read the data because the lock queue is too long to be able to access to the data.
	// If this scenario happens,
	// 1. Then let's flush the data into a file periodically (or once the queue is empty (if it happens))
	// 2. Read the file directly when a read query is coming
	// Like that we have two different ways to read and write the data.
	mutex sync.Mutex
}

func (d *db) GetMetric(name string) *v1.Metric {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	return d.metrics[name]
}

func (d *db) ListMetrics() map[string]*v1.Metric {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	return d.metrics
}

func (d *db) EnqueueMetricList(metrics []string) {
	d.metricsQueue <- metrics
}

func (d *db) EnqueueUsage(usages map[string]*v1.MetricUsage) {
	d.usageQueue <- usages
}

func (d *db) watchMetricsQueue() {
	for _metrics := range d.metricsQueue {
		d.mutex.Lock()
		for _, metricName := range _metrics {
			if _, ok := d.metrics[metricName]; !ok {
				d.metrics[metricName] = &v1.Metric{}
			}
		}
		d.mutex.Unlock()
	}
}

func (d *db) watchUsageQueue() {
	for data := range d.usageQueue {
		d.mutex.Lock()
		for metricName := range data {
			if _, ok := d.metrics[metricName]; !ok {
				logrus.Debugf("metric_name %q is used but it's not found by the metric collector", metricName)
			} else {
				// TODO need to merge the data
			}
		}
		d.mutex.Unlock()
	}
}
