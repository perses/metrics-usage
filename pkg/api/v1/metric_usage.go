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

package v1

import (
	"github.com/perses/common/set"
	"github.com/perses/perses/pkg/model/api/v1/common"
)

type RuleUsage struct {
	PromLink   string `json:"prom_link"`
	GroupName  string `json:"group_name"`
	Name       string `json:"name"`
	Expression string `json:"expression"`
}

type DashboardUsage struct {
	ID   string `json:"uid"`
	Name string `json:"title"`
	URL  string `json:"url"`
}

type MetricUsage struct {
	Dashboards     set.Set[DashboardUsage] `json:"dashboards,omitempty"`
	RecordingRules set.Set[RuleUsage]      `json:"recordingRules,omitempty"`
	AlertRules     set.Set[RuleUsage]      `json:"alertRules,omitempty"`
}

func MergeUsage(old, new *MetricUsage) *MetricUsage {
	if old == nil {
		return new
	}
	if new == nil {
		return old
	}
	return &MetricUsage{
		Dashboards:     set.Merge(old.Dashboards, new.Dashboards),
		AlertRules:     set.Merge(old.AlertRules, new.AlertRules),
		RecordingRules: set.Merge(old.RecordingRules, new.RecordingRules),
	}
}

type Metric struct {
	Labels set.Set[string] `json:"labels,omitempty"`
	Usage  *MetricUsage    `json:"usage,omitempty"`
}

type PartialMetric struct {
	Usage           *MetricUsage    `json:"usage,omitempty"`
	MatchingMetrics set.Set[string] `json:"matchingMetrics,omitempty"`
	MatchingRegexp  *common.Regexp  `json:"matchingRegexp,omitempty"`
}
