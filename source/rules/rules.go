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

package rules

import (
	"context"

	"github.com/perses/common/async"
	"github.com/perses/metrics-usage/config"
	"github.com/perses/metrics-usage/database"
	modelAPIV1 "github.com/perses/metrics-usage/pkg/api/v1"
	"github.com/perses/metrics-usage/pkg/client"
	"github.com/perses/metrics-usage/utils"
	"github.com/perses/metrics-usage/utils/prometheus"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/sirupsen/logrus"
)

func NewCollector(db database.Database, cfg *config.RulesCollector) (async.SimpleTask, error) {
	promClient, err := prometheus.NewClient(cfg.HTTPClient)
	if err != nil {
		return nil, err
	}
	var metricUsageClient client.Client
	if cfg.MetricUsageClient != nil {
		metricUsageClient, err = client.New(*cfg.MetricUsageClient)
		if err != nil {
			return nil, err
		}
	}
	return &rulesCollector{
		promClient:        promClient,
		db:                db,
		metricUsageClient: metricUsageClient,
		promURL:           cfg.HTTPClient.URL.String(),
	}, nil
}

type rulesCollector struct {
	async.SimpleTask
	promClient        v1.API
	db                database.Database
	metricUsageClient client.Client
	promURL           string
}

func (c *rulesCollector) Execute(ctx context.Context, _ context.CancelFunc) error {
	result, err := c.promClient.Rules(ctx)
	if err != nil {
		logrus.WithError(err).Error("Failed to get rules")
		return nil
	}
	metricUsage := extractMetricUsageFromRules(result.Groups, c.promURL)
	if len(metricUsage) > 0 {
		if c.metricUsageClient != nil {
			// In this case, that means we have to send the data to a remote server.
			if sendErr := c.metricUsageClient.Usage(metricUsage); sendErr != nil {
				logrus.WithError(sendErr).Error("Failed to send usage metric")
			}
		} else {
			c.db.EnqueueUsage(metricUsage)
		}
	}
	return nil
}

func (c *rulesCollector) String() string {
	return "rules collector"
}

func extractMetricUsageFromRules(ruleGroups []v1.RuleGroup, source string) map[string]*modelAPIV1.MetricUsage {
	metricUsage := make(map[string]*modelAPIV1.MetricUsage)
	for _, ruleGroup := range ruleGroups {
		for _, rule := range ruleGroup.Rules {
			switch v := rule.(type) {
			case v1.RecordingRule:
				metricNames, parserErr := prometheus.ExtractMetricNamesFromPromQL(v.Query)
				if parserErr != nil {
					logrus.WithError(parserErr).Errorf("Failed to extract metric name for the ruleGroup %q and the recordingRule %q", ruleGroup.Name, v.Name)
					continue
				}
				populateUsage(metricUsage,
					metricNames,
					modelAPIV1.RuleUsage{
						PromLink:   source,
						GroupName:  ruleGroup.Name,
						Name:       v.Name,
						Expression: v.Query,
					},
					false,
				)
			case v1.AlertingRule:
				metricNames, parserErr := prometheus.ExtractMetricNamesFromPromQL(v.Query)
				if parserErr != nil {
					logrus.WithError(parserErr).Errorf("Failed to extract metric name for the ruleGroup %q and the alertingRule %q", ruleGroup.Name, v.Name)
					continue
				}
				populateUsage(metricUsage,
					metricNames,
					modelAPIV1.RuleUsage{
						PromLink:   source,
						GroupName:  ruleGroup.Name,
						Name:       v.Name,
						Expression: v.Query,
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

func populateUsage(metricUsage map[string]*modelAPIV1.MetricUsage, metricNames []string, item modelAPIV1.RuleUsage, isAlertingRules bool) {
	for _, metricName := range metricNames {
		if usage, ok := metricUsage[metricName]; ok {
			if isAlertingRules {
				usage.AlertRules = utils.InsertIfNotPresent(usage.AlertRules, item)
			} else {
				usage.RecordingRules = utils.InsertIfNotPresent(usage.RecordingRules, item)
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
