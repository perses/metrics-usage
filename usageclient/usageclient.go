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

package usageclient

import (
	"github.com/perses/metrics-usage/database"
	modelAPIV1 "github.com/perses/metrics-usage/pkg/api/v1"
	"github.com/perses/metrics-usage/pkg/client"
	"github.com/sirupsen/logrus"
)

type Client struct {
	DB                database.Database
	MetricUsageClient client.Client
	Logger            *logrus.Entry
}

func (c *Client) SendUsage(metricUsage map[string]*modelAPIV1.MetricUsage, invalidMetricUsage map[string]*modelAPIV1.MetricUsage) {
	c.sendMetricUsage(metricUsage)
	c.sendInvalidMetricUsage(invalidMetricUsage)
}

func (c *Client) sendMetricUsage(usage map[string]*modelAPIV1.MetricUsage) {
	if len(usage) == 0 {
		return
	}
	if c.MetricUsageClient != nil {
		// In this case, that means we have to send the data to a remote server.
		if sendErr := c.MetricUsageClient.Usage(usage); sendErr != nil {
			c.Logger.WithError(sendErr).Error("Failed to send usage for metric")
		}
	} else {
		c.DB.EnqueueUsage(usage)
	}
}

func (c *Client) sendInvalidMetricUsage(usage map[string]*modelAPIV1.MetricUsage) {
	if len(usage) == 0 {
		return
	}
	if c.MetricUsageClient != nil {
		// In this case, that means we have to send the data to a remote server.
		if sendErr := c.MetricUsageClient.InvalidMetricsUsage(usage); sendErr != nil {
			c.Logger.WithError(sendErr).Error("Failed to send usage for invalid_metric")
		}
	} else {
		c.DB.EnqueueInvalidMetricsUsage(usage)
	}
}
