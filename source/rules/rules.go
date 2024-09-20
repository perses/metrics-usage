package rules

import (
	"context"

	"github.com/perses/common/async"
	"github.com/perses/metrics-usage/config"
	"github.com/perses/metrics-usage/database"
	"github.com/perses/metrics-usage/utils/prometheus"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/sirupsen/logrus"
)

func NewCollector(db database.Database, cfg config.RulesCollector) (async.SimpleTask, error) {
	promURL := cfg.PrometheusClient.URL.String()
	promClient, err := prometheus.NewClient(cfg.PrometheusClient.TLSConfig, promURL)
	if err != nil {
		return nil, err
	}
	return &rulesCollector{
		client:  promClient,
		db:      db,
		promURL: promURL,
	}, nil
}

type rulesCollector struct {
	async.SimpleTask
	client  v1.API
	db      database.Database
	promURL string
}

func (c *rulesCollector) Execute(ctx context.Context, _ context.CancelFunc) error {
	result, err := c.client.Rules(ctx)
	if err != nil {
		logrus.WithError(err).Error("Failed to get rules")
		return nil
	}
	metricUsage := prometheus.ExtractMetricUsageFromRules(result.Groups, c.promURL)
	if len(metricUsage) > 0 {
		c.db.EnqueueUsage(metricUsage)
	}
	return nil
}

func (c *rulesCollector) String() string {
	return "rules collector"
}
