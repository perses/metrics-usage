Configuration
=============

Metrics-usage is configured via command-line flags and a configuration file

## Flags available

```bash
  -config string
        Path to the yaml configuration file for the api. Configuration can be overridden when using the environment variable
  -log.level string
        log level. Possible value: panic, fatal, error, warning, info, debug, trace (default "info")
  -log.method-trace
        include the calling method as a field in the log. Can be useful to see immediately where the log comes from
  -pprof
    	Enable pprof
  -web.hide-port
        If true, it won t be print on stdout the port listened to receive the HTTP request
  -web.listen-address string
        The address to listen on for HTTP requests, web interface and telemetry. (default ":8080")
  -web.telemetry-path string
        Path under which to expose metrics. (default "/metrics")
```

Example:

```bash
metrics-usage --config=./config.yaml --log.method-trace
```

## Configuration File

### Definition

The file is written in YAML format, defined by the scheme described below. Brackets indicate that a parameter is optional.

Generic placeholders are defined as follows:

* `<boolean>`: a boolean that can take the values `true` or `false`
* `<duration>`: a duration matching the regular expression `((([0-9]+)y)?(([0-9]+)w)?(([0-9]+)d)?(([0-9]+)h)?(([0-9]+)m)?(([0-9]+)s)?(([0-9]+)ms)?|0)`, e.g. `1d`, `1h30m`, `5m`, `10s`
* `<filename>`: a valid path in the current working directory
* `<path>`: a valid URL path
* `<int>`: an integer value
* `<secret>`: a regular string that is a secret, such as a password
* `<string>`: a regular string

```yaml
[ metric_collector: <Metric_Collector config> ]
[ rules_collectors: 
  - <Rule_Collector config> ]
[ perses_collector: <Perses_Collector config> ]
[ grafana_collector: <Grafana_Collector config> ]
```

### Metric_Collector Config

```yaml
[ enable: <boolean> | default=false ]
[ period: <duration> | default="12h" ]
http_client: <HTTPClient config>
```

### Rules_Collector Config

```yaml
[ enable: <boolean> | default=false ]
[ period: <duration> | default="12h" ]
  
# It is a client to send the metrics usage to a remote metrics_usage server.
[ metric_usage_client: <HTTPClient config> ]

# The prometheus client used to retrieve the rules
prometheus_client: <HTTPClient config>
```

### Perses_Collector Config

```yaml
[ enable: <boolean> | default=false ]
[ period: <duration> | default="12h" ]
# It is a client to send the metrics usage to a remote metrics_usage server.
[ metric_usage_client: <HTTPClient config> ]

# the Perses client used to retrieve the dashboards
perses_client:
  url: <string>
  [ tls_config: <TLS Config> ]
  auth:
    basic_auth:
      username: <string>
      [ password: <string> ]
      [ password_file: <string> ]
    [ oauth: < Oauth Config> ]
```

### Grafana_Collector Config

```yaml
[ enable: <boolean> | default=false ]
[ period: <duration> | default="12h" ]
# It is a client to send the metrics usage to a remote metrics_usage server.
[ metric_usage_client: <HTTPClient config> ]

# the Grafana client used to retrieve the dashboards
grafana_client: < HTTPClient config>
```

### TLS Config

```yaml
# CA certificate to validate API server certificate with. At most one of ca and ca_file is allowed.
[ ca: <secret> ]
[ caFile: <filename> ]

# Certificate and key for client cert authentication to the server.
# At most one of cert and cert_file is allowed.
# At most one of key and key_file is allowed.
[ cert: <secret> ]
[ certFile: <filename> ]
[ key: <secret> ]
[ keyFile: <filename> ]

# ServerName extension to indicate the name of the server.
# https://tools.ietf.org/html/rfc4366#section-3.1
[ serverName: <string> ]

# Disable validation of the server certificate.
[ insecureSkipVerify: <boolean> | default = false ]
```

### HTTPClient Config

```yaml
url: <string>
[ oauth: < Oauth Config> ]
[ basic_auth: <BasicAuth Config> ]
[ authorization: <Authorization Config> ]
[ tls_config: < TLS Config> ]
```

### BasicAuth config

```yaml
username: <string>
[ password: <string> ]
[ passwordFile: <filename> ]
```

### Authorization Config

```yaml
[ type: <string> | default = "Bearer" ]

  # The HTTP credentials like a Bearer token
[ credentials: <string> ]
[ credentialsFile: <filename> ]
```

### Oauth Config

```yaml
# ClientID is the application's ID.
client_id: <string>

# ClientSecret is the application's secret.
client_secret: <string>

# TokenURL is the resource server's token endpoint URL. This is a constant specific to each server.
token_url: <string>
```
