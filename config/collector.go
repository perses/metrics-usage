package config

import (
	"fmt"
	"time"

	"github.com/perses/perses/pkg/client/config"
	"github.com/perses/perses/pkg/model/api/v1/common"
	"github.com/perses/perses/pkg/model/api/v1/secret"
	"github.com/prometheus/common/model"
)

const (
	defaultMetricCollectorPeriodDuration = 12 * time.Hour
)

type HTTPClient struct {
	URL       *common.URL       `yaml:"url"`
	TLSConfig *secret.TLSConfig `yaml:"tls_config"`
}

type MetricCollector struct {
	Enable           bool           `yaml:"enable"`
	Period           model.Duration `yaml:"period,omitempty"`
	PrometheusClient HTTPClient     `yaml:"prometheus_client"`
}

func (c *MetricCollector) Verify() error {
	if !c.Enable {
		return nil
	}
	if c.Period <= 0 {
		c.Period = model.Duration(defaultMetricCollectorPeriodDuration)
	}
	if c.PrometheusClient.URL == nil {
		return fmt.Errorf("missing Prometheus URL for the metric collector")
	}
	return nil
}

type RulesCollector struct {
	Enable           bool           `yaml:"enable"`
	Period           model.Duration `yaml:"period,omitempty"`
	PrometheusClient HTTPClient     `yaml:"prometheus_client"`
}

func (c *RulesCollector) Verify() error {
	if !c.Enable {
		return nil
	}
	if c.Period <= 0 {
		c.Period = model.Duration(defaultMetricCollectorPeriodDuration)
	}
	if c.PrometheusClient.URL == nil {
		return fmt.Errorf("missing Prometheus URL for the rules collector")
	}
	return nil
}

type PersesCollector struct {
	Enable     bool                    `yaml:"enable"`
	Period     model.Duration          `yaml:"period,omitempty"`
	HTTPClient config.RestConfigClient `yaml:"http_client"`
}

func (c *PersesCollector) Verify() error {
	if !c.Enable {
		return nil
	}
	if c.Period <= 0 {
		c.Period = model.Duration(defaultMetricCollectorPeriodDuration)
	}
	if c.HTTPClient.URL == nil {
		return fmt.Errorf("missing Rest URL for the perses collector")
	}
	return nil
}
