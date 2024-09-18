package database

import (
	"sync"

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
	queue chan map[string]*v1.Metric
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

func (d *db) Enqueue(metrics map[string]*v1.Metric) {
	d.queue <- metrics
}

func (d *db) watchQueue() {
	for data := range d.queue {
		d.mutex.Lock()
		for metricName, metric := range data {
			if _, ok := d.metrics[metricName]; !ok {
				d.metrics[metricName] = metric
			} else {
				// TODO need to merge the data
			}
		}
		d.mutex.Unlock()
	}
}
