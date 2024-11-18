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

package database

import (
	"testing"

	"github.com/perses/perses/pkg/model/api/v1/common"
	"github.com/stretchr/testify/assert"
)

func newRegexp(re string) *common.Regexp {
	r := common.MustNewRegexp(re)
	return &r
}

func TestGenerateRegexp(t *testing.T) {
	tests := []struct {
		title         string
		invalidMetric string
		result        *common.Regexp
	}{
		{
			title:         "metric equal to a variable",
			invalidMetric: "${metric}",
			result:        nil,
		},
		{
			title:         "metric with variable a suffix",
			invalidMetric: "otelcol_exporter_enqueue_failed_log_records${suffix}",
			result:        newRegexp(`^otelcol_exporter_enqueue_failed_log_records.+$`),
		},
		{
			title:         "metric with multiple variable 1",
			invalidMetric: "${foo}${bar}${john}${doe}",
			result:        nil,
		},
		{
			title:         "metric with multiple variable 2",
			invalidMetric: "prefix_${foo}${bar}:collection_${collection}_suffix:${john}${doe}",
			result:        newRegexp(`^prefix_.+:collection_.+_suffix:.+$`),
		},
		{
			title:         "metric no variable",
			invalidMetric: "otelcol_receiver_.+",
			result:        newRegexp(`^otelcol_receiver_.+$`),
		},
	}

	for _, test := range tests {
		t.Run(test.title, func(t *testing.T) {
			re, err := generateRegexp(test.invalidMetric)
			assert.NoError(t, err)
			assert.Equal(t, test.result, re)
		})
	}
}
