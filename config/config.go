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

import (
	"fmt"
	"time"

	"github.com/perses/common/config"
	"github.com/prometheus/common/model"
)

const defaultFlushPeriod = time.Minute * 5

type Database struct {
	// Define if the database is stored in a file or in memory
	InMemory *bool `yaml:"in_memory,omitempty"`
	// In case the database is stored in a file, then the path to a JSON file must be defined
	Path string `yaml:"path,omitempty"`
	// FlushPeriod defines the frequency the system will flush the data into the JSON file
	FlushPeriod model.Duration `yaml:"flush_period,omitempty"`
}

func (d *Database) Verify() error {
	var inMemory = true
	if d.InMemory == nil {
		d.InMemory = &inMemory
	}
	if *d.InMemory {
		return nil
	}
	if len(d.Path) == 0 {
		return fmt.Errorf("database path is required")
	}
	if d.FlushPeriod == 0 {
		d.FlushPeriod = model.Duration(defaultFlushPeriod)
	}
	return nil
}

type Config struct {
	Database         Database           `yaml:"database"`
	MetricCollector  MetricCollector    `yaml:"metric_collector,omitempty"`
	RulesCollectors  []*RulesCollector  `yaml:"rules_collectors,omitempty"`
	LabelsCollectors []*LabelsCollector `yaml:"labels_collectors,omitempty"`
	PersesCollector  PersesCollector    `yaml:"perses_collector,omitempty"`
	GrafanaCollector GrafanaCollector   `yaml:"grafana_collector,omitempty"`
}

func Resolve(configFile string) (Config, error) {
	c := Config{}
	return c, config.NewResolver[Config]().
		SetConfigFile(configFile).
		SetEnvPrefix("METRICS_USAGE").
		Resolve(&c).
		Verify()
}
