package prometheus

import (
	"github.com/perses/metrics-usage/config"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/promql/parser"
)

func NewClient(cfg config.HTTPClient) (v1.API, error) {
	httpClient, err := config.NewHTTPClient(cfg)
	if err != nil {
		return nil, err
	}
	promHTTPClient, err := api.NewClient(api.Config{
		Address: cfg.URL.String(),
		Client:  httpClient,
	})
	if err != nil {
		return nil, err
	}
	return v1.NewAPI(promHTTPClient), nil
}

func ExtractMetricNamesFromPromQL(query string) ([]string, error) {
	expr, err := parser.ParseExpr(query)
	if err != nil {
		return nil, err
	}
	var result []string
	parser.Inspect(expr, func(node parser.Node, _ []parser.Node) error {
		if n, ok := node.(*parser.VectorSelector); ok {
			// The metric name is only present when the node is a VectorSelector.
			// Then if the vector has the for metric_name{labelName="labelValue"}, then .Name is set.
			// Otherwise, we need to look at the labelName __name__ to find it.
			// Note: we will need to change this rule with Prometheus 3.0
			if n.Name != "" {
				result = append(result, n.Name)
				return nil
			}
			for _, m := range n.LabelMatchers {
				if m.Name == labels.MetricName {
					result = append(result, m.Value)
					return nil
				}
			}
		}
		return nil
	})
	return result, nil
}
