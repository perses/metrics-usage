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

package labels

import (
	"context"
	"time"

	"github.com/perses/common/async"
	"github.com/perses/metrics-usage/config"
	"github.com/perses/metrics-usage/database"
	"github.com/perses/metrics-usage/pkg/client"
	"github.com/perses/metrics-usage/utils/prometheus"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/sirupsen/logrus"
)

func NewCollector(db database.Database, cfg *config.LabelsCollector) (async.SimpleTask, error) {
	promClient, err := prometheus.NewClient(cfg.HTTPClient)
	if err != nil {
		return nil, err
	}
	var metricUsageClient client.Client
	if cfg.MetricUsageClient != nil {
		metricUsageClient, err = client.New(*cfg.MetricUsageClient)
		if err != nil {
			return nil, err
		}
	}
	return &labelCollector{
		promClient:        promClient,
		db:                db,
		metricUsageClient: metricUsageClient,
		period:            cfg.Period,
		logger:            logrus.StandardLogger().WithField("collector", "labels"),
	}, nil
}

type labelCollector struct {
	async.SimpleTask
	promClient        v1.API
	db                database.Database
	metricUsageClient client.Client
	period            model.Duration
	logger            *logrus.Entry
}

func (c *labelCollector) Execute(ctx context.Context, _ context.CancelFunc) error {
	now := time.Now()
	start := now.Add(time.Duration(-c.period))
	labelValues, _, err := c.promClient.LabelValues(ctx, model.MetricNameLabel, nil, start, now)
	if err != nil {
		c.logger.WithError(err).Error("failed to query metrics")
		return nil
	}
	result := make(map[string][]string)
	for _, metricName := range labelValues {
		c.logger.Debugf("querying Prometheus to get label names for metric %s", metricName)
		labels, _, queryErr := c.promClient.LabelNames(ctx, []string{string(metricName)}, start, now)
		if queryErr != nil {
			c.logger.WithError(queryErr).Errorf("failed to query labels for the metric %q", metricName)
			continue
		}
		result[string(metricName)] = removeLabelName(labels)
	}
	if len(result) > 0 {
		if c.metricUsageClient != nil {
			// In this case, that means we have to send the data to a remote server.
			if sendErr := c.metricUsageClient.Labels(result); sendErr != nil {
				c.logger.WithError(sendErr).Error("Failed to send labels name")
			}
		} else {
			c.db.EnqueueLabels(result)
		}
	}
	return nil
}

func (c *labelCollector) String() string {
	return "labels collector"
}

func removeLabelName(labels []string) []string {
	for i, label := range labels {
		if label == model.MetricNameLabel {
			return append(labels[:i], labels[i+1:]...)
		}
	}
	return labels
}
