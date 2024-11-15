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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractMetricNameWithVariable(t *testing.T) {
	tests := []struct {
		title  string
		expr   string
		result []string
	}{
		{
			title:  "single variable at the end of a metric",
			expr:   "sum(${metric:value}(otelcol_processor_batch_batch_size_trigger_send${suffix}{processor=~\"$processor\",job=\"$job\"}[$__rate_interval])) by (processor $grouping)",
			result: []string{"otelcol_processor_batch_batch_size_trigger_send${suffix}"},
		},
		{
			title:  "multiple variable in metric",
			expr:   "sum(${metric:value}(${foo}otelcol_processor_${bar}batch_batch_size_trigger_send${suffix}{processor=~\"$processor\",job=\"$job\"}[$__rate_interval])) by (processor $grouping)",
			result: []string{"${foo}otelcol_processor_${bar}batch_batch_size_trigger_send${suffix}"},
		},
		{
			title:  "variable in label",
			expr:   "rate(tomcat_requestprocessor_received_bytes{$onlyAddsExporter,phase=~\"$phase\",instance=~\"$instance\"}[5m])",
			result: []string{"tomcat_requestprocessor_received_bytes"},
		},
	}
	for _, test := range tests {
		t.Run(test.title, func(t *testing.T) {
			result := extractMetricNameWithVariable(test.expr)
			assert.Equal(t, test.result, result)
		})
	}
}
