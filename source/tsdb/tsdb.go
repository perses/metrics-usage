// Copyright 2025 The Perses Authors
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

package tsdb

import (
	"context"
	"fmt"
	"time"

	"github.com/perses/common/async"
	"github.com/perses/metrics-usage/config"
	"github.com/perses/metrics-usage/database"
	modelAPIV1 "github.com/perses/metrics-usage/pkg/api/v1"
	promUtils "github.com/perses/metrics-usage/utils/prometheus"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/sirupsen/logrus"
)

func NewCollector(db database.Database, cfg config.TSDBCollector) (async.SimpleTask, error) {
	promClient, err := promUtils.NewClient(cfg.HTTPClient)
	if err != nil {
		return nil, err
	}

	logger := logrus.StandardLogger().WithField("collector", "tsdb")
	url := cfg.HTTPClient.URL.URL
	if cfg.PublicURL != nil {
		url = cfg.PublicURL.URL
	}

	return &tsdbCollector{
		db:         db,
		promURL:    url.String(),
		promClient: promClient,
		logger:     logger,
		period:     cfg.Period,
		limit:      cfg.Limit,
	}, nil
}

type tsdbCollector struct {
	async.SimpleTask
	db         database.Database
	promClient v1.API
	promURL    string
	logger     *logrus.Entry
	period     model.Duration
	limit      int
}

func (c *tsdbCollector) Execute(ctx context.Context, _ context.CancelFunc) error {
	result, err := c.promClient.TSDB(ctx, v1.WithLimit(uint64(c.limit)))
	if err != nil {
		c.logger.WithError(err).Error("Failed to get TSDB statistics")
		return nil
	}

	c.logger.Infof("TSDB statistics retrieved successfuly")

	now := time.Now()
	start := now.Add(time.Duration(-c.period))
	stats := map[string]*modelAPIV1.MetricStatistics{}
	for _, v := range result.SeriesCountByMetricName {
		stats[v.Name] = &modelAPIV1.MetricStatistics{
			SeriesCount: v.Value,
			Period:      uint64(time.Duration(c.period) / time.Second),
		}

		metricMatcher := []string{fmt.Sprintf("%s", v.Name)}
		labels, _, err := c.promClient.LabelNames(ctx, metricMatcher, start, now)
		if err != nil {
			c.logger.WithError(err).Errorf("failed to query labels for metric %q", v.Name)
			return nil
		}

		for _, label := range labels {
			if label == model.MetricNameLabel {
				continue
			}

			values, _, err := c.promClient.LabelValues(ctx, label, metricMatcher, start, now)
			if err != nil {
				c.logger.WithError(err).Errorf("failed to query label values for label %q metric %q", label, v.Name)
				return nil
			}

			stats[v.Name].LabelValueCountByLabelName = append(
				stats[v.Name].LabelValueCountByLabelName,
				modelAPIV1.LabelCount{Name: label, Value: uint64(len(values))},
			)
		}
	}

	c.db.EnqueueMetricStatistics(stats)
	return nil
}

func (c *tsdbCollector) String() string {
	return "TSDB statistics collector"
}
