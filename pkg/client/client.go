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

func (c *client) url(ep string) *url.URL {
	p := path.Join(c.endpoint.Path, ep)
	u := *c.endpoint
	u.Path = p

	return &u
}
