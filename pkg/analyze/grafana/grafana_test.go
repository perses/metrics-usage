// Copyright 2024 The Perses Authors
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
	"encoding/json"
	"os"
	"testing"

	modelAPIV1 "github.com/perses/metrics-usage/pkg/api/v1"
	"github.com/stretchr/testify/assert"
)

func unmarshalDashboard(path string) (*SimplifiedDashboard, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	result := &SimplifiedDashboard{}
	return result, json.Unmarshal(data, result)
}

func TestAnalyze(t *testing.T) {
	tests := []struct {
		name          string
		dashboardFile string
		resultMetrics []string
		resultErrs    []*modelAPIV1.LogError
	}{
		{
			name:          "from/to variables",
			dashboardFile: "tests/d1.json",
			resultMetrics: []string{"run", "service_color"},
		},
		{
			name:          "use static variable",
			dashboardFile: "tests/d2.json",
			resultMetrics: []string{"gacms_svc_elapsed_time_seconds_bucket"},
		},
		{
			name:          "variable replace order",
			dashboardFile: "tests/d3.json",
			resultMetrics: []string{"probe_success"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dashboard, err := unmarshalDashboard(tt.dashboardFile)
			if err != nil {
				t.Fatal(err)
			}
			metrics, errs := Analyze(dashboard)
			assert.Equal(t, tt.resultMetrics, metrics)
			assert.Equal(t, tt.resultErrs, errs)
		})
	}
}
