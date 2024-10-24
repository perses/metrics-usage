package grafana

import (
	"context"
	"encoding/json"

	"github.com/go-openapi/strfmt"
	grafanaapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/client/search"
	grafanaModels "github.com/grafana/grafana-openapi-client-go/models"
	"github.com/perses/common/async"
	"github.com/perses/metrics-usage/config"
	"github.com/perses/metrics-usage/database"
	modelAPIV1 "github.com/perses/metrics-usage/pkg/api/v1"
	"github.com/sirupsen/logrus"
)

func NewCollector(db database.Database, cfg config.GrafanaCollector) (async.SimpleTask, error) {
	httpClient, err := config.NewHTTPClient(cfg.HTTPClient)
	url := cfg.HTTPClient.URL.URL
	if err != nil {
		return nil, err
	}
	transportConfig := &grafanaapi.TransportConfig{
		Host:     url.Host,
		BasePath: grafanaapi.DefaultBasePath,
		Schemes:  []string{url.Scheme},
		Client:   httpClient,
	}
	client := grafanaapi.NewHTTPClientWithConfig(strfmt.Default, transportConfig)
	return &grafanaCollector{
		db:         db,
		grafanaURL: url.String(),
		client:     client,
	}, nil
}

type grafanaCollector struct {
	async.SimpleTask
	db         database.Database
	grafanaURL string
	client     *grafanaapi.GrafanaHTTPAPI
}

func (c *grafanaCollector) Execute(ctx context.Context, _ context.CancelFunc) error {
	hits, err := c.collectAllDashboardUID(ctx)
	if err != nil {
		logrus.WithError(err).Error("failed to collect dashboard UIDs")
		return nil
	}

	metricUsage := make(map[string]*modelAPIV1.MetricUsage)
	for _, h := range hits {
		_, getErr := c.getDashboard(h.UID)
		if getErr != nil {
			logrus.WithError(getErr).Errorf("failed to get dashboard %q with UID %q", h.Title, h.UID)
			continue
		}
	}
	if len(metricUsage) > 0 {
		c.db.EnqueueUsage(metricUsage)
	}
	return nil
}

func (c *grafanaCollector) getDashboard(uid string) (*simplifiedDashboard, error) {
	response, err := c.client.Dashboards.GetDashboardByUID(uid)
	if err != nil {
		return nil, err
	}
	rowData, err := json.Marshal(response.Payload.Dashboard)
	if err != nil {
		return nil, err
	}
	result := &simplifiedDashboard{}
	return result, json.Unmarshal(rowData, &result)
}

func (c *grafanaCollector) collectAllDashboardUID(ctx context.Context) ([]*grafanaModels.Hit, error) {
	var currentPage int64 = 1
	var result []*grafanaModels.Hit
	searchOk := true
	// value based on the comment from the code here: https://github.com/grafana/grafana-openapi-client-go/blob/9d96c2007bd8c89981630106307c8764e3d02747/client/search/search_parameters.go#L151
	searchType := "dash-db"

	for searchOk {
		nextPageResult, err := c.client.Search.Search(&search.SearchParams{
			Context: ctx,
			Type:    &searchType,
			Page:    &currentPage,
		})
		if err != nil {
			return nil, err
		}
		searchOk = nextPageResult.IsSuccess() && len(nextPageResult.Payload) > 0
		currentPage++
		if searchOk {
			result = append(result, nextPageResult.Payload...)
		}
	}
	return result, nil
}

func (c *grafanaCollector) String() string {
	return "grafana collector"
}
