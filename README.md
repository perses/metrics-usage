Metrics Usage
============

[![build](https://github.com/perses/metrics-usage/workflows/ci/badge.svg)](https://github.com/perses/metrics-usage/actions?query=workflow%3Aci)
[![Go Report Card](https://goreportcard.com/badge/github.com/perses/metrics-usage)](https://goreportcard.com/report/github.com/perses/metrics-usage)

This tool analyzes static files - like dashboards and Prometheus alert rules - to track where and how Prometheus metrics are used.

It’s especially helpful for identifying whether metrics are actively used.
Prometheus should ideally not scrape unused metrics to avoid an unnecessary load.

## API exposed

### Metrics

The tool provides an API endpoint, `/api/v1/metrics`, which returns the usage data for each collected metric as shown below:

```json
{
  "node_cpu_seconds_total": {
    "usage": {
      "dashboards": [
        {
          "id": "myinsight/first_demo",
          "name": "first_demo",
          "url": "https://demo.perses.dev/api/v1/projects/myinsight/dashboards/first_demo"
        },
        {
          "id": "myworkshopproject/myfirstdashboard",
          "name": "myfirstdashboard",
          "url": "https://demo.perses.dev/api/v1/projects/myworkshopproject/dashboards/myfirstdashboard"
        },
        {
          "id": "perses/nodeexporterfull",
          "name": "nodeexporterfull",
          "url": "https://demo.perses.dev/api/v1/projects/perses/dashboards/nodeexporterfull"
        },
        {
          "id": "showcase/statchartpanel",
          "name": "statchartpanel",
          "url": "https://demo.perses.dev/api/v1/projects/showcase/dashboards/statchartpanel"
        }
      ],
      "recordingRules": [
        {
          "prom_link": "https://prometheus.demo.do.prometheus.io",
          "group_name": "node-exporter.rules",
          "name": "instance:node_num_cpu:sum",
          "expression": "count without (cpu, mode) (node_cpu_seconds_total{job=\"node\",mode=\"idle\"})"
        },
        {
          "prom_link": "https://prometheus.demo.do.prometheus.io",
          "group_name": "node-exporter.rules",
          "name": "instance:node_cpu_utilisation:rate5m",
          "expression": "1 - avg without (cpu) (sum without (mode) (rate(node_cpu_seconds_total{job=\"node\",mode=~\"idle|iowait|steal\"}[5m])))"
        }
      ],
      "alertRules": [
        {
          "prom_link": "https://prometheus.demo.do.prometheus.io",
          "group_name": "node-exporter",
          "name": "NodeCPUHighUsage",
          "expression": "sum without (mode) (avg without (cpu) (rate(node_cpu_seconds_total{job=\"node\",mode!=\"idle\"}[2m]))) * 100 > 90"
        },
        {
          "prom_link": "https://prometheus.demo.do.prometheus.io",
          "group_name": "node-exporter",
          "name": "NodeSystemSaturation",
          "expression": "node_load1{job=\"node\"} / count without (cpu, mode) (node_cpu_seconds_total{job=\"node\",mode=\"idle\"}) > 2"
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
          "name": "NodeCPUUtilizationHigh",
          "expression": "instance:node_cpu_utilisation:rate5m * 100 > ignoring (severity) node_cpu_utilization_percent_threshold{severity=\"critical\"}"
        }
      ]
    }
  },
  "node_disk_discard_time_seconds_total": {
    "usage": {
      "dashboards": [
        {
          "id": "perses/nodeexporterfull",
          "name": "nodeexporterfull",
          "url": "https://demo.perses.dev/api/v1/projects/perses/dashboards/nodeexporterfull"
        }
      ]
    }
  }
}
```

You can use the following query parameter to filter the list returned:

* **metric_name**: when used, it will trigger a fuzzy match search on the metric_name or exact/regex match if you enable it with `mode` parameter.
* **used**: when used, will return only the metric used or not (depending on if you set this boolean to true or to false). Leave it empty if you want both.
* **mode**: when used change mode for filtering metrics. Three values are available:
  * **exact**: doing exact match (case sensitive) based on the metric name
  * **fuzzy**: doing fuzzy match based on the metric name
  * **regex**: doing regex match based on the metric name
* **merge_partial_metrics**: when used, it will use the data from /api/v1/partial_metrics and merge them here.

### Partial Metrics

The API endpoint `/api/v1/partial_metrics` is exposing the usage for metrics that contains variable or regexp. 

```json
{
  "node_disk_discard_time_.+": {
    "usage": {
      "alertRules": [
        {
          "prom_link": "https://prometheus.demo.do.prometheus.io",
          "group_name": "ansible managed alert rules",
          "name": "NodeCPUUtilizationHigh",
          "expression": "instance:node_cpu_utilisation:rate5m * 100 > ignoring (severity) node_cpu_utilization_percent_threshold{severity=\"critical\"}"
        }
      ]
    }
  },
  "node_cpu_utilization_${instance}": {
    "usage": {
      "dashboards": [
        {
          "id": "perses/nodeexporterfull",
          "name": "nodeexporterfull",
          "url": "https://demo.perses.dev/api/v1/projects/perses/dashboards/nodeexporterfull"
        }
      ]
    }
  }
}
```

### Pending Usage

The API endpoint `/api/v1/pending_usages` is exposing usage associated to metrics that has not yet been associated to the metrics available on the endpoint `/api/v1/metrics`. 

It's even possible usage is never associated as the metric doesn't exist anymore.

## Different way to deploy it

### Central instance

Metrics Usage can be configured as a central instance, which collects data from multiple sources in a stateful setup.

![Architecture overview](docs/architecture/central_architecture_usage.svg)

### Sidecar Container for Rules Collection

In setups with numerous rules, central data collection may become impractical due to the volume. Instead, you can deploy Metrics Usage as a sidecar container, configured to push data to a central instance.

![Architecture overview](docs/architecture/sidecar_rules_usage.svg)

## Available Collectors

Metrics Usage offers various collectors for obtaining metric usage data:

### Prometheus Metric Collector

This collector retrieves a list of metrics over a specified period and stores them for association with usage data from other collectors.

#### Configuration

> Refer to the complete configuration [here](./docs/configuration.md#metric_collector-config)

Example:

```yaml
metric_collector:
  enable: true
  prometheus_client:
    url: "https://prometheus.demo.do.prometheus.io"
```

### Prometheus Rule Collector

This collector retrieves Prometheus rule groups using the HTTP API and extracts metrics from alerting & recording rules.

Multiple rule collectors can be configured for different Prometheus/Thanos instances.

#### Configuration

> Refer to the complete configuration [here](./docs/configuration.md#rules_collector-config)

Example: 

```yaml
rules_collectors:
  - enable: true
    prometheus_client:
      url: "https://prometheus.demo.do.prometheus.io"
```

### Perses Collector

This collector fetches dashboards from Perses via its HTTP API, extracting metrics used in variables and panels.

#### Configuration

> Refer to the complete configuration [here](./docs/configuration.md#perses_collector-config)

Example:

```yaml
perses_collector:
  enable: true
  perses_client:
    url: "https://demo.perses.dev"
```

### Grafana Collector

This collector fetches dashboards from Grafana via its HTTP API, extracting metrics used in the panels.

#### Configuration

> Refer to the complete configuration [here](./docs/configuration.md#grafana_collector-config)

Example:

```yaml
grafana_collector:
  enable: true
  grafana_client:
    url: "https//demo.grafana.dev"
```

## Install

There are several ways of installing Metrics Usage:

### Precompiled binaries

Download precompiled binaries from the [GitHub releases page](https://github.com/perses/metrics-usage/releases). It is recommended to use the latest release available.

### Docker images

Docker images are available on [Docker Hub](https://hub.docker.com/r/persesdev/metrics-usage).

To try it out with Docker:

```bash
docker run --name metrics-usage -d -p 127.0.0.1:8080:8080 persesdev/metrics-usage
```

### Building from source

> To build from source, you’ll need Go [version 1.23 or higher](https://golang.org/doc/install).

Start by cloning the repository:

```bash
git clone https://github.com/perses/metrics-usage.git
cd metrics-usage
```

Then build the web assets and Metrics Usage itself with:

```bash
make build
./bin/metrics-usage --config=your_config.yml
```
