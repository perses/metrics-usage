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

package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"

	"github.com/perses/metrics-usage/config"
	modelAPIV1 "github.com/perses/metrics-usage/pkg/api/v1"
)

type Client interface {
	Usage(map[string]*modelAPIV1.MetricUsage) error
	Labels(map[string][]string) error
}

func New(cfg config.HTTPClient) (Client, error) {
	httpClient, err := config.NewHTTPClient(cfg)
	if err != nil {
		return nil, err
	}
	return &client{
		endpoint:   cfg.URL.URL,
		httpClient: httpClient,
	}, nil
}

type client struct {
	endpoint   *url.URL
	httpClient *http.Client
}

func (c *client) Usage(metrics map[string]*modelAPIV1.MetricUsage) error {
	data, err := json.Marshal(metrics)
	if err != nil {
		return err
	}
	body := bytes.NewBuffer(data)
	resp, err := c.httpClient.Post(c.url("/api/v1/metrics").String(), "application/json", body)
	if err != nil {
		return err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode > http.StatusPartialContent {
		return fmt.Errorf("when sending metrics usage, unexpected status code: %d", resp.StatusCode)
	}
	return nil
}

func (c *client) InvalidMetricsUsage(metrics map[string]*modelAPIV1.MetricUsage) error {
	data, err := json.Marshal(metrics)
	if err != nil {
		return err
	}
	body := bytes.NewBuffer(data)
	resp, err := c.httpClient.Post(c.url("/api/v1/invalid_metrics").String(), "application/json", body)
	if err != nil {
		return err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode > http.StatusPartialContent {
		return fmt.Errorf("when sending metrics usage, unexpected status code: %d", resp.StatusCode)
	}
	return nil
}

func (c *client) Labels(labels map[string][]string) error {
	data, err := json.Marshal(labels)
	if err != nil {
		return err
	}
	body := bytes.NewBuffer(data)
	resp, err := c.httpClient.Post(c.url("/api/v1/labels").String(), "application/json", body)
	if err != nil {
		return err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode > http.StatusPartialContent {
		return fmt.Errorf("when sending label names, unexpected status code: %d", resp.StatusCode)
	}
	return nil
}

func (c *client) url(ep string) *url.URL {
	p := path.Join(c.endpoint.Path, ep)
	u := *c.endpoint
	u.Path = p

	return &u
}
