package prometheus

import (
	"net"
	"net/http"
	"time"

	modelAPIV1 "github.com/perses/metrics-usage/pkg/api/v1"
	"github.com/perses/perses/pkg/model/api/v1/secret"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/promql/parser"
	"github.com/sirupsen/logrus"
)

const connectionTimeout = 30 * time.Second

func NewClient(tlsConfig *secret.TLSConfig, url string) (v1.API, error) {
	tlsCfg, err := secret.BuildTLSConfig(tlsConfig)
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
		Address:      url,
		RoundTripper: roundTripper,
	})
	if err != nil {
		return nil, err
	}
	return v1.NewAPI(httpClient), nil
}

func ExtractMetricUsageFromRules(ruleGroups []v1.RuleGroup, source string) map[string]*modelAPIV1.MetricUsage {
	metricUsage := make(map[string]*modelAPIV1.MetricUsage)
	for _, ruleGroup := range ruleGroups {
		for _, rule := range ruleGroup.Rules {
			switch v := rule.(type) {
			case v1.RecordingRule:
				metricNames, parserErr := extractMetricNamesFromPromQL(v.Query)
				if parserErr != nil {
					logrus.WithError(parserErr).Errorf("Failed to extract metric name for the ruleGroup %q and the recordingRule %q", ruleGroup.Name, v.Name)
					continue
				}
				populateUsage(metricUsage,
					metricNames,
					modelAPIV1.RuleUsage{
						PromLink:  source,
						GroupName: ruleGroup.Name,
						Name:      v.Name,
					},
					false,
				)
			case v1.AlertingRule:
				metricNames, parserErr := extractMetricNamesFromPromQL(v.Query)
				if parserErr != nil {
					logrus.WithError(parserErr).Errorf("Failed to extract metric name for the ruleGroup %q and the alertingRule %q", ruleGroup.Name, v.Name)
					continue
				}
				populateUsage(metricUsage,
					metricNames,
					modelAPIV1.RuleUsage{
						PromLink:  source,
						GroupName: ruleGroup.Name,
						Name:      v.Name,
					},
					true,
				)
			default:
				logrus.Debugf("unknown rule type %s", v)
			}
		}
	}
	return metricUsage
}

func extractMetricNamesFromPromQL(query string) ([]string, error) {
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

func populateUsage(metricUsage map[string]*modelAPIV1.MetricUsage, metricNames []string, item modelAPIV1.RuleUsage, isAlertingRules bool) {
	for _, metricName := range metricNames {
		if usage, ok := metricUsage[metricName]; ok {
			if isAlertingRules {
				usage.AlertRules = insertIfNotPresent(usage.AlertRules, item)
			} else {
				usage.RecordingRules = insertIfNotPresent(usage.RecordingRules, item)
			}
		} else {
			u := &modelAPIV1.MetricUsage{}
			if isAlertingRules {
				u.AlertRules = []modelAPIV1.RuleUsage{item}
			} else {
				u.RecordingRules = []modelAPIV1.RuleUsage{item}
			}
			metricUsage[metricName] = u
		}
	}
}

func insertIfNotPresent(slice []modelAPIV1.RuleUsage, item modelAPIV1.RuleUsage) []modelAPIV1.RuleUsage {
	for _, s := range slice {
		if item == s {
			return slice
		}
	}
	return append(slice, item)
}
