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

package config

import "github.com/perses/common/config"

type Config struct {
	MetricCollector  MetricCollector   `yaml:"metric_collector,omitempty"`
	RulesCollectors  []*RulesCollector `yaml:"rules_collectors,omitempty"`
	PersesCollector  PersesCollector   `yaml:"perses_collector,omitempty"`
	GrafanaCollector GrafanaCollector  `yaml:"grafana_collector,omitempty"`
}

func Resolve(configFile string) (Config, error) {
	c := Config{}
	return c, config.NewResolver[Config]().
		SetConfigFile(configFile).
		SetEnvPrefix("METRICS_USAGE").
		Resolve(&c).
		Verify()
}
