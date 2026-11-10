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

package metric

import (
	"encoding/json"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/perses/common/set"
	"github.com/perses/metrics-usage/database"
	v1 "github.com/perses/metrics-usage/pkg/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeDB struct {
	database.Database
	metrics             map[string]*v1.Metric
	usage               map[string]*v1.MetricUsage
	partialMetricsUsage map[string]*v1.MetricUsage
}

func (f *fakeDB) GetMetric(name string) *v1.Metric {
	return f.metrics[name]
}

func (f *fakeDB) DeleteMetric(name string) bool {
	if _, ok := f.metrics[name]; !ok {
		return false
	}
	delete(f.metrics, name)
	return true
}

func (f *fakeDB) ListMetrics() (map[string]*v1.Metric, error) {
	return f.metrics, nil
}

func (f *fakeDB) EnqueueUsage(usages map[string]*v1.MetricUsage) {
	f.usage = usages
}

func (f *fakeDB) EnqueuePartialMetricsUsage(usages map[string]*v1.MetricUsage) {
	f.partialMetricsUsage = usages
}

func serve(db database.Database, method string, target string, body string) *httptest.ResponseRecorder {
	e := echo.New()
	NewAPI(db).RegisterRoute(e)
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func decodeMessage(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var response map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	return response["message"]
}

func TestGetAndDeleteMetric(t *testing.T) {
	tests := []struct {
		title        string
		method       string
		path         string
		expectedCode int
		expectedBody string
		expectExists bool
	}{
		{
			title:        "get a metric",
			method:       http.MethodGet,
			path:         "/api/v1/metrics/up",
			expectedCode: http.StatusOK,
			expectedBody: `{"labels":["job"]}`,
			expectExists: true,
		},
		{
			title:        "get an unknown metric",
			method:       http.MethodGet,
			path:         "/api/v1/metrics/unknown",
			expectedCode: http.StatusNotFound,
			expectedBody: `{"message":"Not Found"}`,
			expectExists: true,
		},
		{
			title:        "delete a metric",
			method:       http.MethodDelete,
			path:         "/api/v1/metrics/up",
			expectedCode: http.StatusNoContent,
			expectExists: false,
		},
		{
			title:        "delete an unknown metric",
			method:       http.MethodDelete,
			path:         "/api/v1/metrics/unknown",
			expectedCode: http.StatusNotFound,
			expectedBody: `{"message":"Not Found"}`,
			expectExists: true,
		},
	}
	for _, test := range tests {
		t.Run(test.title, func(t *testing.T) {
			db := &fakeDB{metrics: map[string]*v1.Metric{"up": {Labels: set.New("job")}}}
			rec := serve(db, test.method, test.path, "")

			assert.Equal(t, test.expectedCode, rec.Code)
			if test.expectedBody == "" {
				assert.Empty(t, rec.Body.String())
			} else {
				assert.JSONEq(t, test.expectedBody, rec.Body.String())
			}
			assert.Equal(t, test.expectExists, db.GetMetric("up") != nil)
		})
	}
}

func TestListMetrics(t *testing.T) {
	usage := &v1.MetricUsage{Dashboards: set.New(v1.DashboardUsage{ID: "perses/node", Name: "node", URL: "https://demo.perses.dev"})}
	tests := []struct {
		title           string
		query           url.Values
		expectedMetrics []string
	}{
		{
			title:           "no filter",
			expectedMetrics: []string{"node_cpu_seconds_total", "node_memory_free_bytes", "up"},
		},
		{
			title:           "used metrics",
			query:           url.Values{"used": {"true"}},
			expectedMetrics: []string{"node_cpu_seconds_total"},
		},
		{
			title:           "unused metrics",
			query:           url.Values{"used": {"false"}},
			expectedMetrics: []string{"node_memory_free_bytes", "up"},
		},
		{
			title:           "exact match by default",
			query:           url.Values{"metric_name": {"up"}},
			expectedMetrics: []string{"up"},
		},
		{
			title:           "regex match",
			query:           url.Values{"metric_name": {"^node_.+"}, "mode": {"regex"}},
			expectedMetrics: []string{"node_cpu_seconds_total", "node_memory_free_bytes"},
		},
		{
			title:           "fuzzy match on the unused metrics",
			query:           url.Values{"metric_name": {"node"}, "mode": {"fuzzy"}, "used": {"false"}},
			expectedMetrics: []string{"node_memory_free_bytes"},
		},
	}
	for _, test := range tests {
		t.Run(test.title, func(t *testing.T) {
			db := &fakeDB{metrics: map[string]*v1.Metric{
				"node_cpu_seconds_total": {Usage: usage},
				"node_memory_free_bytes": {},
				"up":                     {},
			}}
			rec := serve(db, http.MethodGet, "/api/v1/metrics?"+test.query.Encode(), "")

			require.Equal(t, http.StatusOK, rec.Code)
			var metrics map[string]*v1.Metric
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &metrics))
			assert.Equal(t, test.expectedMetrics, slices.Sorted(maps.Keys(metrics)))
		})
	}
}

func TestBadRequest(t *testing.T) {
	tests := []struct {
		title           string
		method          string
		target          string
		body            string
		expectedMessage string
	}{
		{
			title:           "invalid match mode",
			method:          http.MethodGet,
			target:          "/api/v1/metrics?metric_name=up&mode=unknown",
			expectedMessage: "invalid match mode: unknown. Allowed: exact, fuzzy or regex",
		},
		{
			title:           "invalid query parameter",
			method:          http.MethodGet,
			target:          "/api/v1/metrics?used=maybe",
			expectedMessage: `strconv.ParseBool: parsing "maybe": invalid syntax`,
		},
		{
			title:           "invalid metrics usage",
			method:          http.MethodPost,
			target:          "/api/v1/metrics",
			body:            `{"up":x}`,
			expectedMessage: "invalid character 'x' looking for beginning of value",
		},
		{
			title:           "invalid partial metrics usage",
			method:          http.MethodPost,
			target:          "/api/v1/partial_metrics",
			body:            `{"node_cpu_${mode}":{"dashboards":"perses/node"}}`,
			expectedMessage: "cannot unmarshal string",
		},
	}
	for _, test := range tests {
		t.Run(test.title, func(t *testing.T) {
			db := &fakeDB{}
			rec := serve(db, test.method, test.target, test.body)

			assert.Equal(t, http.StatusBadRequest, rec.Code)
			// The message contains the reason of the failure, whatever the format of the error returned by echo.
			assert.Contains(t, decodeMessage(t, rec), test.expectedMessage)
			assert.Nil(t, db.usage)
			assert.Nil(t, db.partialMetricsUsage)
		})
	}
}

func TestPushUsageRoutes(t *testing.T) {
	body := `{"node_cpu_${mode}":{"dashboards":[{"uid":"perses/node","title":"node","url":"https://demo.perses.dev"}]}}`
	tests := []struct {
		title         string
		path          string
		expectUsage   bool
		expectPartial bool
	}{
		{
			title:       "push metrics usage",
			path:        "/api/v1/metrics",
			expectUsage: true,
		},
		{
			title:         "push partial metrics usage",
			path:          "/api/v1/partial_metrics",
			expectPartial: true,
		},
	}
	for _, test := range tests {
		t.Run(test.title, func(t *testing.T) {
			db := &fakeDB{}
			rec := serve(db, http.MethodPost, test.path, body)

			assert.Equal(t, http.StatusAccepted, rec.Code)
			assert.Equal(t, test.expectUsage, db.usage != nil)
			assert.Equal(t, test.expectPartial, db.partialMetricsUsage != nil)
		})
	}
}
