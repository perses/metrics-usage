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

package config

import (
	"context"
	"encoding/base64"
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
	URL           *common.URL           `yaml:"url"`
	OAuth         *config.OAuth         `yaml:"oauth,omitempty"`
	BasicAuth     *secret.BasicAuth     `yaml:"basic_auth,omitempty"`
	Authorization *secret.Authorization `yaml:"authorization,omitempty"`
	TLSConfig     *secret.TLSConfig     `yaml:"tls_config,omitempty"`
}

func NewHTTPClient(cfg HTTPClient) (*http.Client, error) {
	roundTripper, err := config.NewRoundTripper(connectionTimeout, cfg.TLSConfig)
	if err != nil {
		return nil, err
	}
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, &http.Client{
		Transport: roundTripper,
		Timeout:   connectionTimeout,
	})
	if cfg.OAuth != nil {
		oauthConfig := &clientcredentials.Config{
			ClientID:     cfg.OAuth.ClientID,
			ClientSecret: cfg.OAuth.ClientSecret,
			TokenURL:     cfg.OAuth.TokenURL,
			Scopes:       cfg.OAuth.Scopes,
			AuthStyle:    cfg.OAuth.AuthStyle,
		}
		return oauthConfig.Client(ctx), nil
	}
	if cfg.BasicAuth != nil {
		c := oauth2.Config{}
		password, getPasswordErr := cfg.BasicAuth.GetPassword()
		if getPasswordErr != nil {
			return nil, getPasswordErr
		}
		return c.Client(ctx, &oauth2.Token{
			AccessToken: base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", cfg.BasicAuth.Username, password))),
			TokenType:   "basic",
		}), nil
	}
	if cfg.Authorization != nil {
		c := oauth2.Config{}
		credential, getCredentialErr := cfg.Authorization.GetCredentials()
		if getCredentialErr != nil {
			return nil, getCredentialErr
		}
		return c.Client(ctx, &oauth2.Token{
			AccessToken: credential,
			TokenType:   cfg.Authorization.Type,
		}), nil
	}
	return &http.Client{
		Transport: roundTripper,
		Timeout:   connectionTimeout,
	}, nil
}

type MetricCollector struct {
	Enable     bool           `yaml:"enable"`
	Period     model.Duration `yaml:"period,omitempty"`
	HTTPClient HTTPClient     `yaml:"http_client"`
}

func (c *MetricCollector) Verify() error {
	if !c.Enable {
		return nil
	}
	if c.Period <= 0 {
		c.Period = model.Duration(defaultMetricCollectorPeriodDuration)
	}
	if c.HTTPClient.URL == nil {
		return fmt.Errorf("missing Prometheus URL for the metric collector")
	}
	return nil
}

type RulesCollector struct {
	Enable bool           `yaml:"enable"`
	Period model.Duration `yaml:"period,omitempty"`
	// MetricUsageClient is a client to send the metrics usage to a remote metrics_usage server.
	MetricUsageClient *HTTPClient `yaml:"metric_usage_client,omitempty"`
	// RetryToGetRules is the number of retries the collector will do to get the rules from Prometheus before actually failing.
	// Between each retry, the collector will wait first 10 seconds, then 20 seconds, then 30 seconds ...etc.
	RetryToGetRules uint       `yaml:"retry_to_get_rules,omitempty"`
	HTTPClient      HTTPClient `yaml:"prometheus_client"`
}

func (c *RulesCollector) Verify() error {
	if !c.Enable {
		return nil
	}
	if c.Period <= 0 {
		c.Period = model.Duration(defaultMetricCollectorPeriodDuration)
	}
	if c.RetryToGetRules == 0 {
		c.RetryToGetRules = 3
	}
	if c.HTTPClient.URL == nil {
		return fmt.Errorf("missing Prometheus URL for the rules collector")
	}
	if c.MetricUsageClient != nil && c.MetricUsageClient.URL == nil {
		return fmt.Errorf("missing Metrics Usage URL for the rules collector")
	}
	return nil
}

type PersesCollector struct {
	Enable            bool                    `yaml:"enable"`
	Period            model.Duration          `yaml:"period,omitempty"`
	MetricUsageClient *HTTPClient             `yaml:"metric_usage_client,omitempty"`
	HTTPClient        config.RestConfigClient `yaml:"perses_client"`
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
	if c.MetricUsageClient != nil && c.MetricUsageClient.URL == nil {
		return fmt.Errorf("missing Metrics Usage URL for the rules collector")
	}
	return nil
}

type GrafanaCollector struct {
	Enable            bool           `yaml:"enable"`
	Period            model.Duration `yaml:"period,omitempty"`
	MetricUsageClient *HTTPClient    `yaml:"metric_usage_client,omitempty"`
	HTTPClient        HTTPClient     `yaml:"grafana_client"`
}

func (c *GrafanaCollector) Verify() error {
	if !c.Enable {
		return nil
	}
	if c.Period <= 0 {
		c.Period = model.Duration(defaultMetricCollectorPeriodDuration)
	}
	if c.HTTPClient.URL == nil {
		return fmt.Errorf("missing Rest URL for the perses collector")
	}
	if c.MetricUsageClient != nil && c.MetricUsageClient.URL == nil {
		return fmt.Errorf("missing Metrics Usage URL for the rules collector")
	}
	return nil
}
