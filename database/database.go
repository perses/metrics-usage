package database

import (
	v1 "github.com/perses/metrics-usage/pkg/api/v1"
)

type Database interface {
	GetMetric(name string) *v1.Metric
	ListMetrics() map[string]*v1.Metric
	Enqueue(metrics map[string]*v1.Metric)
}

func New() Database {
	d := &db{
		Database: nil,
		metrics:  make(map[string]*v1.Metric),
		queue:    make(chan map[string]*v1.Metric, 250),
	}

	go d.watchQueue()
	return d
}

type db struct {
	Database
	metrics map[string]*v1.Metric
	// queue is the way to send data to write in the database.
	// There will be no other way to write in it.
	// Doing that allows us to accept more HTTP requests to write data and to delay the actual writing.
	// Also, we shouldn't need to lock the access to the database as there will be only one writer and multiple readers.
	queue chan map[string]*v1.Metric
}

func (d *db) GetMetric(name string) *v1.Metric {
	return d.metrics[name]
}

func (d *db) ListMetrics() map[string]*v1.Metric {
	return d.metrics
}

func (d *db) Enqueue(metrics map[string]*v1.Metric) {
	d.queue <- metrics
}

func (d *db) watchQueue() {
	for metrics := range d.queue {
		for metricName, metric := range metrics {
			if _, ok := d.metrics[metricName]; !ok {
				d.metrics[metricName] = metric
			} else {
				// TODO need to merge the data
			}
		}
	}
}
