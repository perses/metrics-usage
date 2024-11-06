package grafana

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func unmarshalDashboard(path string) (*simplifiedDashboard, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	result := &simplifiedDashboard{}
	return result, json.Unmarshal(data, result)
}

func TestExtractMetrics(t *testing.T) {
	tests := []struct {
		name          string
		dashboardFile string
		resultMetrics []string
		resultErrs    []logError
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
			metrics, errs := extractMetrics(dashboard)
			assert.Equal(t, tt.resultMetrics, metrics)
			assert.Equal(t, tt.resultErrs, errs)
		})
	}
}
