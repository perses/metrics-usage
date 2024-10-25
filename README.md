Metric Usage
============

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

```yaml
metric_collector:
  enable: true
  prometheus_client:
    url: "https://prometheus.demo.do.prometheus.io"
```

### Prometheus Rule Collector

This collector gets the Prometheus Rules Group using the HTTP API. Then it extracts the metric used in the alert rule or in the recording rule.

#### Configuration

```yaml
rules_collector:
  enable: true
  prometheus_client:
    url: "https://prometheus.demo.do.prometheus.io"
```

### Perses Collector

This collector gets the list of dashboards using the HTTP API of Perses. Then it extracts the metric used in the variables and in the different panels.

#### Configuration

```yaml
perses_collector:
  enable: true
  http_client:
    url: "https://demo.perses.dev"
```

### Grafana Collector

This collector gets the list of dashboards using the HTTP API of Grafana. Then it extracts the metric used in the different panels. 
Extraction from variable still needs to be done.

#### Configuration

```yaml
grafana_collector:
  enable: true
  http_client:
    url: "https//demo.grafana.dev"
```
