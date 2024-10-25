package config

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/perses/perses/pkg/client/config"
	"github.com/perses/perses/pkg/model/api/v1/common"
	"github.com/perses/perses/pkg/model/api/v1/secret"
	"github.com/prometheus/common/model"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

const (
	defaultMetricCollectorPeriodDuration = 12 * time.Hour
	connectionTimeout                    = 30 * time.Second
)

type HTTPClient struct {
	URL       *common.URL       `yaml:"url"`
	Oauth     *config.Oauth     `yaml:"oauth,omitempty"`
	TLSConfig *secret.TLSConfig `yaml:"tls_config,omitempty"`
}

func NewHTTPClient(cfg HTTPClient) (*http.Client, error) {
	roundTripper, err := config.NewRoundTripper(connectionTimeout, cfg.TLSConfig)
	if err != nil {
		return nil, err
	}
	if cfg.Oauth != nil {
		oauthConfig := &clientcredentials.Config{
			ClientID:     cfg.Oauth.ClientID,
			ClientSecret: cfg.Oauth.ClientSecret,
			TokenURL:     cfg.Oauth.TokenURL,
			Scopes:       cfg.Oauth.Scopes,
			AuthStyle:    cfg.Oauth.AuthStyle,
		}
		ctx := context.WithValue(context.Background(), oauth2.HTTPClient, &http.Client{
			Transport: roundTripper,
			Timeout:   connectionTimeout,
		})
		return oauthConfig.Client(ctx), nil
	}
	return &http.Client{
		Transport: roundTripper,
		Timeout:   connectionTimeout,
	}, nil
}

type MetricCollector struct {
	Enable           bool           `yaml:"enable"`
	Period           model.Duration `yaml:"period,omitempty"`
	PrometheusClient HTTPClient     `yaml:"http_client"`
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
	PrometheusClient HTTPClient     `yaml:"http_client"`
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

type GrafanaCollector struct {
	Enable     bool           `yaml:"enable"`
	Period     model.Duration `yaml:"period,omitempty"`
	HTTPClient HTTPClient     `yaml:"http_client"`
}
