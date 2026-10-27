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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/perses/metrics-usage/database"
	v1 "github.com/perses/metrics-usage/pkg/api/v1"
	"github.com/stretchr/testify/assert"
)

type fakeDB struct {
	database.Database
	usage               map[string]*v1.MetricUsage
	partialMetricsUsage map[string]*v1.MetricUsage
}

func (f *fakeDB) EnqueueUsage(usages map[string]*v1.MetricUsage) {
	f.usage = usages
}

func (f *fakeDB) EnqueuePartialMetricsUsage(usages map[string]*v1.MetricUsage) {
	f.partialMetricsUsage = usages
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
			e := echo.New()
			NewAPI(db).RegisterRoute(e)

			req := httptest.NewRequest(http.MethodPost, test.path, strings.NewReader(body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusAccepted, rec.Code)
			assert.Equal(t, test.expectUsage, db.usage != nil)
			assert.Equal(t, test.expectPartial, db.partialMetricsUsage != nil)
		})
	}
}
