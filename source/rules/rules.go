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

package rules

import (
	"context"
	"time"

	"github.com/perses/common/async"
	"github.com/perses/metrics-usage/config"
	"github.com/perses/metrics-usage/database"
	"github.com/perses/metrics-usage/pkg/analyze/prometheus"
	"github.com/perses/metrics-usage/pkg/client"
	"github.com/perses/metrics-usage/usageclient"
	promUtils "github.com/perses/metrics-usage/utils/prometheus"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/sirupsen/logrus"
)

func NewCollector(db database.Database, cfg *config.RulesCollector) (async.SimpleTask, error) {
	promClient, err := promUtils.NewClient(cfg.HTTPClient)
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
	logger := logrus.StandardLogger().WithField("collector", "rules")
	return &rulesCollector{
		promClient: promClient,
		metricUsageClient: &usageclient.Client{
			DB:                db,
			MetricUsageClient: metricUsageClient,
			Logger:            logger,
		},
		promURL: cfg.HTTPClient.URL.String(),
		logger:  logger,
		retry:   cfg.RetryToGetRules,
	}, nil
}

type rulesCollector struct {
	async.SimpleTask
	promClient        v1.API
	metricUsageClient *usageclient.Client
	promURL           string
	logger            *logrus.Entry
	retry             uint
}

func (c *rulesCollector) Execute(ctx context.Context, _ context.CancelFunc) error {
	result, err := c.getRules(ctx)
	if err != nil {
		c.logger.WithError(err).Error("Failed to get rules")
		return nil
	}
	metricsUsage, partialMetricsUsage, errs := prometheus.Analyze(result.Groups, c.promURL)
	for _, logErr := range errs {
		logErr.Log(c.logger)
	}
	c.logger.Infof("%d metrics usage has been collected", len(metricsUsage))
	c.logger.Infof("%d metrics containing regexp or variable has been collected", len(partialMetricsUsage))
	c.metricUsageClient.SendUsage(metricsUsage, partialMetricsUsage)
	return nil
}

func (c *rulesCollector) String() string {
	return "rules collector"
}

func (c *rulesCollector) getRules(ctx context.Context) (v1.RulesResult, error) {
	waitDuration := 10 * time.Second
	retry := c.retry
	doRetry := true
	var err error
	var result v1.RulesResult
	for doRetry && retry > 0 {
		result, err = c.promClient.Rules(ctx)
		if err != nil {
			doRetry = true
			retry--
			c.logger.WithError(err).Debug("Failed to get rules, retrying...")
			time.Sleep(waitDuration)
			waitDuration = waitDuration + 10*time.Second
		} else {
			c.logger.Infof("successfuly get the rules")
			doRetry = false
		}
	}
	return result, err
}
