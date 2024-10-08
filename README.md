Metric Usage
============

This tool is used to analyze static files such as dashboard, Prometheus alert rules to find the usage of the Prometheus metrics.

It can be useful to have an idea where the metrics are used and if they are used. Certainly, if they are not used, you shouldn't let Prometheus scrap it.

## Available Collectors

There is a various way to collect the metric usage, here the complete list of the available collectors:

### Prometheus Metric Collector

This collector gets the list of metrics for a defined period of time. This list is then stored in the system waiting to associate them with their usage find by the other collectors.

### Prometheus Rule Collector

This collector gets the Prometheus Rules Group using the HTTP API. Then it extracts the metric used in the alert rule or in the recording rule.

### Perses Collector

This collector gets the list of dashboards using the HTTP API of Perses. Then it extracts the metric used in the variables and in the different panels.
