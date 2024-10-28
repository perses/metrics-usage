Metric Usage
============

[![build](https://github.com/perses/metrics-usage/workflows/ci/badge.svg)](https://github.com/perses/metrics-usage/actions?query=workflow%3Aci)
[![Go Report Card](https://goreportcard.com/badge/github.com/perses/metrics-usage)](https://goreportcard.com/report/github.com/perses/metrics-usage)

This tool is used to analyze static files such as dashboard, Prometheus alert rules to find the usage of the Prometheus metrics.

It can be useful to have an idea where the metrics are used and if they are used. Certainly, if they are not used, you shouldn't let Prometheus scrap it.

This tool provides an API that can be used to get the usage for each metrics collected. Requesting `/api/v1/metrics` will give you something like that:

```json
{
  "node_cpu_seconds_total": {
    "usage": {
      "dashboards": [
        "https://demo.perses.dev/api/v1/projects/myinsight/dashboards/first_demo",
        "https://demo.perses.dev/api/v1/projects/myworkshopproject/dashboards/myfirstdashboard",
        "https://demo.perses.dev/api/v1/projects/perses/dashboards/nodeexporterfull",
        "https://demo.perses.dev/api/v1/projects/showcase/dashboards/statchartpanel"
      ],
      "recordingRules": [
        {
          "prom_link": "https://prometheus.demo.do.prometheus.io",
          "group_name": "node-exporter.rules",
          "name": "instance:node_num_cpu:sum"
        },
        {
          "prom_link": "https://prometheus.demo.do.prometheus.io",
          "group_name": "node-exporter.rules",
          "name": "instance:node_cpu_utilisation:rate5m"
        }
      ],
      "alertRules": [
        {
          "prom_link": "https://prometheus.demo.do.prometheus.io",
          "group_name": "node-exporter",
          "name": "NodeCPUHighUsage"
        },
        {
          "prom_link": "https://prometheus.demo.do.prometheus.io",
          "group_name": "node-exporter",
          "name": "NodeSystemSaturation"
        }
      ]
    }
  },
  "node_cpu_utilization_percent_threshold": {
    "usage": {
      "alertRules": [
        {
          "prom_link": "https://prometheus.demo.do.prometheus.io",
          "group_name": "ansible managed alert rules",
          "name": "NodeCPUUtilizationHigh"
        }
      ]
    }
  },
  "node_disk_discard_time_seconds_total": {
    "usage": {
      "dashboards": [
        "https://demo.perses.dev/api/v1/projects/perses/dashboards/nodeexporterfull"
      ]
    }
  }
}
```

## Available Collectors

There is a various way to collect the metric usage, here the complete list of the available collectors:

### Prometheus Metric Collector

This collector gets the list of metrics for a defined period of time. This list is then stored in the system waiting to associate them with their usage find by the other collectors.

#### Configuration

See the doc for the complete configuration [here](./docs/configuration.md#metric_collector-config)

Example:

```yaml
metric_collector:
  enable: true
  prometheus_client:
    url: "https://prometheus.demo.do.prometheus.io"
```

### Prometheus Rule Collector

This collector gets the Prometheus Rules Group using the HTTP API. Then it extracts the metric used in the alert rule or in the recording rule.

You define a set of rule collectors as we are assuming rules can com from various Prometheus / Thanos rulers.

#### Configuration

See the doc for the complete configuration [here](./docs/configuration.md#rules_collector-config)

Example: 

```yaml
rules_collectors:
  - enable: true
    prometheus_client:
      url: "https://prometheus.demo.do.prometheus.io"
```

### Perses Collector

This collector gets the list of dashboards using the HTTP API of Perses. Then it extracts the metric used in the variables and in the different panels.

#### Configuration

See the doc for the complete configuration [here](./docs/configuration.md#perses_collector-config)

Example:

```yaml
perses_collector:
  enable: true
  http_client:
    url: "https://demo.perses.dev"
```

### Grafana Collector

This collector gets the list of dashboards using the HTTP API of Grafana. Then it extracts the metric used in the different panels. 

> [!IMPORTANT]
> Extraction from variable still needs to be done.

#### Configuration

See the doc for the complete configuration [here](./docs/configuration.md#grafana_collector-config)

Example:

```yaml
grafana_collector:
  enable: true
  http_client:
    url: "https//demo.grafana.dev"
```

## Install

There are various ways of installing Metrics-Usage.

### Precompiled binaries

Precompiled binaries for released versions are available in
the [GitHub release](https://github.com/perses/metrics-usage/releases). Using the latest release binary is the recommended way
of installing Metrics-Usage.

### Docker images

Docker images are available on [Docker Hub](https://hub.docker.com/r/persesdev/metrics-usage).

You can launch a Metrics-Usage container for trying it out with:

```bash
docker run --name metrics-usage -d -p 127.0.0.1:8080:8080 persesdev/metrics-usage
```

### Building from source

To build Metrics-Usage from source code, You need:

- Go [version 1.23 or greater](https://golang.org/doc/install).

Start by cloning the repository:

```bash
git clone https://github.com/perses/metrics-usage.git
cd metrics-usage
```

Then you can use `make build` that would build the web assets and then Metrics-Usage itself.

```bash
make build
./bin/metrics-usage --config=your_config.yml
```
