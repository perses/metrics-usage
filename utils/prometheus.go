package utils

import (
	"net"
	"net/http"
	"time"

	"github.com/perses/perses/pkg/model/api/v1/secret"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

const connectionTimeout = 30 * time.Second

func NewPrometheusClient(tlsConfig *secret.TLSConfig, url string) (v1.API, error) {
	tlsCfg, err := secret.BuildTLSConfig(tlsConfig)
	if err != nil {
		return nil, err
	}
	roundTripper := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   connectionTimeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig:     tlsCfg, // nolint: gas, gosec
	}
	httpClient, err := api.NewClient(api.Config{
		Address:      url,
		RoundTripper: roundTripper,
	})
	if err != nil {
		return nil, err
	}
	return v1.NewAPI(httpClient), nil
}
