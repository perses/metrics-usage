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

package metric

import (
	"context"
	"time"

	"github.com/perses/common/async"
	"github.com/perses/metrics-usage/config"
	"github.com/perses/metrics-usage/database"
	"github.com/perses/metrics-usage/utils/prometheus"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/sirupsen/logrus"
)

func NewCollector(db database.Database, cfg config.MetricCollector) (async.SimpleTask, error) {
	promClient, err := prometheus.NewClient(cfg.HTTPClient)
	if err != nil {
		return nil, err
	}
	return &metricCollector{
		client: promClient,
		db:     db,
		period: cfg.Period,
	}, nil
}

type metricCollector struct {
	async.SimpleTask
	client v1.API
	db     database.Database
	period model.Duration
}

func (c *metricCollector) Execute(ctx context.Context, _ context.CancelFunc) error {
	now := time.Now()
	start := now.Add(time.Duration(-c.period))
	labelValues, _, err := c.client.LabelValues(ctx, "__name__", nil, start, now)
	if err != nil {
		logrus.WithError(err).Error("failed to query metrics")
		return nil
	}
	result := make([]string, 0, len(labelValues))
	for _, metricName := range labelValues {
		result = append(result, string(metricName))
	}
	// Finally, send the metric collected to the database; db will take care to store these data properly
	if len(result) > 0 {
		c.db.EnqueueMetricList(result)
	}
	return nil
}

func (c *metricCollector) String() string {
	return "metric collector"
}
