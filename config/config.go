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
