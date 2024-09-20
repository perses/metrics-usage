package rules

import (
	"context"

	"github.com/perses/common/async"
	"github.com/perses/metrics-usage/config"
	"github.com/perses/metrics-usage/database"
	modelAPIV1 "github.com/perses/metrics-usage/pkg/api/v1"
	"github.com/perses/metrics-usage/utils"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/promql/parser"
	"github.com/sirupsen/logrus"
)

func NewCollector(db database.Database, cfg config.RulesCollector) (async.SimpleTask, error) {
	promURL := cfg.PrometheusClient.URL.String()
	promClient, err := utils.NewPrometheusClient(cfg.PrometheusClient.TLSConfig, promURL)
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
	metricUsage := make(map[string]*modelAPIV1.MetricUsage)
	for _, ruleGroup := range result.Groups {
		for _, rule := range ruleGroup.Rules {
			switch v := rule.(type) {
			case v1.RecordingRule:
				metricNames, parserErr := extractMetricName(v.Query)
				if parserErr != nil {
					logrus.WithError(parserErr).Errorf("Failed to extract metric name for the ruleGroup %q and the recordingRule %q", ruleGroup.Name, v.Name)
					continue
				}
				populateUsage(metricUsage,
					metricNames,
					modelAPIV1.RuleUsage{
						PromLink:  c.promURL,
						GroupName: ruleGroup.Name,
						Name:      v.Name,
					},
					false,
				)
			case v1.AlertingRule:
				metricNames, parserErr := extractMetricName(v.Query)
				if parserErr != nil {
					logrus.WithError(parserErr).Errorf("Failed to extract metric name for the ruleGroup %q and the alertingRule %q", ruleGroup.Name, v.Name)
					continue
				}
				populateUsage(metricUsage,
					metricNames,
					modelAPIV1.RuleUsage{
						PromLink:  c.promURL,
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
	if len(metricUsage) > 0 {
		c.db.EnqueueUsage(metricUsage)
	}
	return nil
}

func (c *rulesCollector) String() string {
	return "rules collector"
}

func extractMetricName(query string) ([]string, error) {
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
			// Note that, we will need to change this rule with Prometheus 3.0
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
