package metric

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/perses/common/async"
	"github.com/perses/metrics-usage/config"
	"github.com/perses/metrics-usage/database"
	"github.com/perses/perses/pkg/model/api/v1/secret"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/sirupsen/logrus"
)

const connectionTimeout = 30 * time.Second

func NewCollector(db database.Database, cfg config.MetricCollector) (async.SimpleTask, error) {
	tlsCfg, err := secret.BuildTLSConfig(cfg.PrometheusClient.TLSConfig)
	if err != nil {
		return nil, err
	}
	roundTripper := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   connectionTimeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig:     tlsCfg, // nolint: gas, gosec
	}
	httpClient, err := api.NewClient(api.Config{
		Address:      cfg.PrometheusClient.URL.String(),
		RoundTripper: roundTripper,
	})
	if err != nil {
		return nil, err
	}
	return &metricCollector{
		client: v1.NewAPI(httpClient),
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

func (g *metricCollector) Execute(ctx context.Context, _ context.CancelFunc) error {
	now := time.Now()
	start := now.Add(time.Duration(-g.period))
	labelValues, _, err := g.client.LabelValues(ctx, "__name__", nil, start, now)
	if err != nil {
		logrus.WithError(err).Error("failed to query metrics")
		return nil
	}
	result := make([]string, 0, len(labelValues))
	for _, metricName := range labelValues {
		result = append(result, string(metricName))
	}
	// Finally, send the metric collected to the database; db will take care to store these data properly
	g.db.EnqueueMetricList(result)
	return nil
}

func (g *metricCollector) String() string {
	return "metric collector"
}
