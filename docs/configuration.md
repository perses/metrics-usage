# Configuration

Metrics Usage is configured via command-line flags and a configuration file.

## Flags available

```bash
  -config string
    	Path to the YAML configuration file for the API. Configuration settings can be overridden when using environment variables.
  -log.format string
    	log format. Possible value: text, json (default "text")
  -log.level string
    	log level. Possible value: panic, fatal, error, warning, info, debug, trace (default "info")
  -log.method-trace
    	include the calling method as a field in the log. Can be useful to see immediately where the log comes from
  -pprof
    	Enable pprof
  -web.hide-port
    	If true, it won't be print on stdout the port listened to receive the HTTP request
  -web.listen-address string
    	The address to listen on for HTTP requests, web interface and telemetry. (default ":8080")
  -web.telemetry-path string
    	Path under which to expose metrics. (default "/metrics")
  -web.tls-cert-file string
    	The path to the cert to use for the HTTPS server
  -web.tls-cipher-suites string
    	Comma-separated list of TLS cipher suite names
  -web.tls-key-file string
    	The path to the key to use for the HTTPS server
  -web.tls-max-version string
    	Maximum TLS version (e.g., "1.2", "1.3")
  -web.tls-min-version string
    	Minimum TLS version (e.g., "1.2", "1.3")
```

Example:

```bash
metrics-usage --config=./config.yaml --log.method-trace
```

## Configuration File

Metrics Usage can be configured using a YAML file and/or environment variables.

### Environment variables

You can use both the configuration file and the environment variables. The environment variables override the
configuration from the file.

To get the environment variable corresponding to a setting, join all the keys leading to it in the YAML with `_`, put
the result in uppercase and prefix it with `METRICS_USAGE`. For example, the environment variable corresponding to the
query parser engine is:

```bash
METRICS_USAGE_QUERY_PARSER_ENGINE=metricsql
```

When you have a list of objects, use the index to specify the object you want to configure. For example, to configure
the Prometheus URL of the first rules collector:

```bash
METRICS_USAGE_RULES_COLLECTORS_0_ENABLE=true
METRICS_USAGE_RULES_COLLECTORS_0_PROMETHEUS_CLIENT_URL=https://prometheus.demo.prometheus.io
```

### Definition

The file is written in YAML format, defined by the scheme described below. Brackets indicate that a parameter is optional.

Generic placeholders are defined as follows:

- `<boolean>`: a boolean that can take the values `true` or `false`
- `<duration>`: a duration matching the regular expression `((([0-9]+)y)?(([0-9]+)w)?(([0-9]+)d)?(([0-9]+)h)?(([0-9]+)m)?(([0-9]+)s)?(([0-9]+)ms)?|0)`, e.g. `1d`, `1h30m`, `5m`, `10s`
- `<filename>`: a valid path in the current working directory
- `<int>`: an integer value
- `<secret>`: a regular string that is a secret, such as a password
- `<string>`: a regular string
- `<url>`: a valid URL

```yaml
[ database: <Database config> ]
[ metric_collector: <Metric_Collector config> ]
[ labels_collectors:
  - <Labels_Collector config> ]
[ rules_collectors:
  - <Rules_Collector config> ]
[ perses_collector: <Perses_Collector config> ]
[ grafana_collector: <Grafana_Collector config> ]
[ query_parser: <Query_Parser config> ]
```

### Database Config

```yaml
# It defines if the database is stored in a file or in memory.
[ in_memory: <boolean> | default = true ]

# In case the database is stored in a file, then the path to a JSON file must be defined.
[ path: <filename> ]

# It defines the frequency the system will flush the data into the JSON file.
[ flush_period: <duration> | default = 5m ]

# It is the number of consecutive times the metric collector didn't see a metric before removing it from the database.
[ threshold: <int> | default = 2 ]
```

Example:

```yaml
database:
  in_memory: false
  path: "/data/metrics-usage.json"
  flush_period: 10m
```

### Metric_Collector Config

The metric collector retrieves the list of metrics available in Prometheus. These metrics are the reference used to
associate the usage found by the other collectors.

```yaml
[ enable: <boolean> | default = false ]

# It is the frequency of the collection, as well as the time range used to retrieve the metrics.
[ period: <duration> | default = 12h ]

# The Prometheus client used to retrieve the metrics.
http_client: <HTTPClient config>
```

Example:

```yaml
metric_collector:
  enable: true
  http_client:
    url: "https://prometheus.demo.prometheus.io"
```

### Labels_Collector Config

The labels collector retrieves the label names of each metric available in Prometheus.

```yaml
[ enable: <boolean> | default = false ]

# It is the frequency of the collection, as well as the time range used to retrieve the labels.
[ period: <duration> | default = 12h ]

# It is the number of metrics for which the labels are retrieved in parallel.
[ concurrency: <int> | default = 1 ]

# It is a client to send the labels to a remote Metrics Usage server.
# When not set, the labels are stored in the local database.
[ metric_usage_client: <HTTPClient config> ]

# The Prometheus client used to retrieve the labels.
prometheus_client: <HTTPClient config>
```

Example:

```yaml
labels_collectors:
  - enable: true
    concurrency: 10
    prometheus_client:
      url: "https://prometheus.demo.prometheus.io"
```

### Rules_Collector Config

The rules collector retrieves the rule groups from Prometheus (or any compatible API like Thanos Ruler) and extracts the
metrics used in the alerting and recording rules.

```yaml
[ enable: <boolean> | default = false ]

# It is the frequency of the collection.
[ period: <duration> | default = 12h ]

# It is a client to send the metrics usage to a remote Metrics Usage server.
# When not set, the usage is stored in the local database.
[ metric_usage_client: <HTTPClient config> ]

# It is the number of retries the collector will do to get the rules from Prometheus before actually failing.
# Between each retry, the collector will wait first 10 seconds, then 20 seconds, then 30 seconds ...etc.
[ retry_to_get_rules: <int> | default = 3 ]

# The Prometheus client used to retrieve the rules.
prometheus_client: <HTTPClient config>

# It is the URL used in the usage to link to the Prometheus instance (the field `prom_link`).
# Useful when the URL used to reach Prometheus is not the one the users know, for example a localhost URL in a sidecar.
# When not set, the URL of the prometheus_client is used.
[ public_url: <url> ]
```

Example:

```yaml
rules_collectors:
  - enable: true
    prometheus_client:
      url: "http://localhost:10902"
    public_url: "https://thanos-ruler.example.com"
    metric_usage_client:
      url: "https://metrics-usage.example.com"
```

### Perses_Collector Config

The Perses collector retrieves the dashboards from Perses and extracts the metrics used in the variables and in the
panels.

```yaml
[ enable: <boolean> | default = false ]

# It is the frequency of the collection.
[ period: <duration> | default = 12h ]

# It is a client to send the metrics usage to a remote Metrics Usage server.
# When not set, the usage is stored in the local database.
[ metric_usage_client: <HTTPClient config> ]

# The Perses client used to retrieve the dashboards.
perses_client: <PersesClient config>

# It is the URL used in the usage to link to the dashboards.
# When not set, the URL of the perses_client is used.
[ public_url: <url> ]
```

Example:

```yaml
perses_collector:
  enable: true
  perses_client:
    url: "https://demo.perses.dev"
```

### Grafana_Collector Config

The Grafana collector retrieves the dashboards from Grafana and extracts the metrics used in the panels.

```yaml
[ enable: <boolean> | default = false ]

# It is the frequency of the collection.
[ period: <duration> | default = 12h ]

# It is a client to send the metrics usage to a remote Metrics Usage server.
# When not set, the usage is stored in the local database.
[ metric_usage_client: <HTTPClient config> ]

# The Grafana client used to retrieve the dashboards.
grafana_client: <HTTPClient config>

# It is the URL used in the usage to link to the dashboards.
# When not set, the URL of the grafana_client is used.
[ public_url: <url> ]

# List of datasource types to ignore when extracting metric expressions from dashboard targets.
# Targets with matching datasource types will be skipped during metric extraction.
# Datasource type matching is case-insensitive (e.g., "postgres" matches "POSTGRES").
# Example: ["postgres", "mysql", "influxdb"]
[ ignore_datasource_types: <list of strings> ]

# List of datasource UIDs to ignore when extracting metric expressions from dashboard targets.
# Targets with matching datasource UIDs will be skipped during metric extraction.
# Example: ["bf5b9901-8a28-4501-a0c2-726b4ee3f15d", "another-uid"]
[ ignore_datasource_uids: <list of strings> ]
```

Example:

```yaml
grafana_collector:
  enable: true
  period: 12h
  grafana_client:
    url: "https://grafana.example.com"
    authorization:
      credentialsFile: "/etc/metrics-usage/grafana-token"
  ignore_datasource_types:
    - postgres
    - mysql
  ignore_datasource_uids:
    - bf5b9901-8a28-4501-a0c2-726b4ee3f15d
```

### Query_Parser Config

```yaml
# Engine specifies the query parser engine to use for analyzing PromQL/MetricsQL expressions.
# Valid values are: "promql" (default) or "metricsql"
[ engine: <string> | default = "promql" ]
```

The query parser engine determines how metric names are extracted from query expressions:

- `promql`: Uses the Prometheus PromQL parser (default, backward compatible)
- `metricsql`: Uses the VictoriaMetrics MetricsQL parser, which supports PromQL plus additional MetricsQL extensions

Example:

```yaml
query_parser:
  engine: metricsql
```

### HTTPClient Config

```yaml
url: <url>

# Only one authentication method is used. When several are set, the precedence is: oauth, basic_auth, authorization.
[ oauth: <OAuth config> ]
[ basic_auth: <BasicAuth config> ]
[ authorization: <Authorization config> ]

[ tls_config: <TLS config> ]
```

### PersesClient Config

```yaml
url: <url>

# Credentials of a Perses user, used to log in with the native authentication of Perses.
[ native_auth:
    login: <string>
    password: <secret> ]

[ oauth: <OAuth config> ]
[ basic_auth: <BasicAuth config> ]
[ authorization: <Authorization config> ]

# Use the credentials of a kubeconfig file, when Perses is configured with the Kubernetes authentication.
[ k8s_auth:
    [ kubeconfig: <filename> ] ]

[ tls_config: <TLS config> ]

# Additional HTTP headers sent with every request.
[ headers:
    [ <string>: <string> ... ] ]
```

### BasicAuth Config

```yaml
username: <string>

# At most one of password and passwordFile is allowed.
[ password: <secret> ]
[ passwordFile: <filename> ]
```

### Authorization Config

```yaml
[ type: <string> | default = "Bearer" ]

# The HTTP credentials like a token.
# At most one of credentials and credentialsFile is allowed.
[ credentials: <secret> ]
[ credentialsFile: <filename> ]
```

### OAuth Config

This is the OAuth 2.0 client credentials flow.

```yaml
# ClientID is the application's ID.
clientID: <string>

# ClientSecret is the application's secret.
# At most one of clientSecret and clientSecretFile is allowed, and one of them is required.
[ clientSecret: <secret> ]
[ clientSecretFile: <filename> ]

# TokenURL is the resource server's token endpoint URL. This is a constant specific to each server.
tokenURL: <url>

# Scopes specifies optional requested permissions.
[ scopes:
  - <string> ... ]

# AuthStyle optionally specifies how the endpoint wants the client ID & client secret sent.
# 0: auto-detect (default), 1: in the parameters, 2: in the header (HTTP basic authorization).
[ authStyle: <int> | default = 0 ]
```

### TLS Config

```yaml
# CA certificate to validate API server certificate with. At most one of ca and caFile is allowed.
[ ca: <string> ]
[ caFile: <filename> ]

# Certificate and key for client cert authentication to the server.
# At most one of cert and certFile is allowed.
# At most one of key and keyFile is allowed.
[ cert: <string> ]
[ certFile: <filename> ]
[ key: <secret> ]
[ keyFile: <filename> ]

# ServerName extension to indicate the name of the server.
# https://tools.ietf.org/html/rfc4366#section-3.1
[ serverName: <string> ]

# Disable validation of the server certificate.
[ insecureSkipVerify: <boolean> | default = false ]

# Minimum acceptable TLS version. Accepted values: TLS10 (TLS 1.0), TLS11 (TLS 1.1), TLS12 (TLS 1.2), TLS13 (TLS 1.3).
[ minVersion: <string> | default = TLS12 ]

# Maximum acceptable TLS version. Accepted values: TLS10 (TLS 1.0), TLS11 (TLS 1.1), TLS12 (TLS 1.2), TLS13 (TLS 1.3).
[ maxVersion: <string> | default = TLS13 ]
```
