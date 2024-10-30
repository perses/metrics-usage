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

type RuleUsage struct {
	PromLink  string `json:"prom_link"`
	GroupName string `json:"group_name"`
	Name      string `json:"name"`
}

type MetricUsage struct {
	Dashboards     []string    `json:"dashboards,omitempty"`
	RecordingRules []RuleUsage `json:"recordingRules,omitempty"`
	AlertRules     []RuleUsage `json:"alertRules,omitempty"`
}

type Metric struct {
	Usage *MetricUsage `json:"usage,omitempty"`
}
