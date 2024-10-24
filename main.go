package main

import (
	"flag"
	"time"

	"github.com/perses/common/app"
	"github.com/perses/metrics-usage/config"
	"github.com/perses/metrics-usage/database"
	"github.com/perses/metrics-usage/source/grafana"
	"github.com/perses/metrics-usage/source/metric"
	"github.com/perses/metrics-usage/source/perses"
	"github.com/perses/metrics-usage/source/rules"
	"github.com/sirupsen/logrus"
)

func main() {
	configFile := flag.String("config", "", "Path to the YAML configuration file for the API. Configuration settings can be overridden when using environment variables.")
	pprof := flag.Bool("pprof", false, "Enable pprof")
	flag.Parse()

	// load the config from file or/and from environment
	conf, err := config.Resolve(*configFile)
	if err != nil {
		logrus.WithError(err).Fatalf("error reading configuration from file %q or from environment", *configFile)
	}

	db := database.New()
	runner := app.NewRunner().WithDefaultHTTPServer("metrics_usage")

	if conf.MetricCollector.Enable {
		metricCollectorConfig := conf.MetricCollector
		metricCollector, collectorErr := metric.NewCollector(db, metricCollectorConfig)
		if collectorErr != nil {
			logrus.WithError(collectorErr).Fatal("unable to create the metric collector")
		}
		runner.WithTimerTasks(time.Duration(metricCollectorConfig.Period), metricCollector)
	}

	if conf.RulesCollector.Enable {
		rulesCollectorConfig := conf.RulesCollector
		rulesCollector, collectorErr := rules.NewCollector(db, rulesCollectorConfig)
		if collectorErr != nil {
			logrus.WithError(collectorErr).Fatal("unable to create the rules collector")
		}
		runner.WithTimerTasks(time.Duration(rulesCollectorConfig.Period), rulesCollector)
	}

	if conf.PersesCollector.Enable {
		persesCollectorConfig := conf.PersesCollector
		persesCollector, collectorErr := perses.NewCollector(db, persesCollectorConfig)
		if collectorErr != nil {
			logrus.WithError(collectorErr).Fatal("unable to create the perses collector")
		}
		runner.WithTimerTasks(time.Duration(persesCollectorConfig.Period), persesCollector)
	}

	if conf.GrafanaCollector.Enable {
		grafanaCollectorConfig := conf.GrafanaCollector
		grafanaCollector, collectorErr := grafana.NewCollector(db, grafanaCollectorConfig)
		if collectorErr != nil {
			logrus.WithError(collectorErr).Fatal("unable to create the grafana collector")
		}
		runner.WithTimerTasks(time.Duration(grafanaCollectorConfig.Period), grafanaCollector)
	}

	runner.HTTPServerBuilder().
		ActivatePprof(*pprof).
		APIRegistration(metric.NewAPI(db)).
		APIRegistration(rules.NewAPI(db))
	runner.Start()
}
