// Copyright The Perses Authors
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

package grafana

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/go-openapi/strfmt"
	grafanaapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/client/search"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecute_ReturnErrorWhenSearchFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)

	c := &grafanaCollector{
		grafanaClient: grafanaapi.NewHTTPClientWithConfig(strfmt.Default, &grafanaapi.TransportConfig{
			Host:     serverURL.Host,
			BasePath: grafanaapi.DefaultBasePath,
			Schemes:  []string{serverURL.Scheme},
		}),
		logger: logrus.NewEntry(logrus.StandardLogger()),
	}
	err = c.Execute(context.Background(), func() {})
	var searchErr *search.SearchInternalServerError
	assert.ErrorAs(t, err, &searchErr)
}
