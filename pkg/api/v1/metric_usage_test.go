// Copyright 2025 The Perses Authors
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

package v1

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJSONMarshalMetricUsage(t *testing.T) {
	testSuite := []struct {
		name         string
		usage        *MetricUsage
		expectedJSON string
	}{
		{
			name:         "empty",
			usage:        &MetricUsage{},
			expectedJSON: `{}`,
		},
		{
			name: "AlertRules",
			usage: &MetricUsage{
				AlertRules: NewSet(RuleUsage{Name: "foo"}, RuleUsage{Name: "bar"}),
			},
			expectedJSON: `{"alertRules":[{"prom_link":"","group_name":"","name":"foo","expression":""},{"prom_link":"","group_name":"","name":"bar","expression":""}]}`,
		},
	}
	for _, test := range testSuite {
		t.Run(test.name, func(t *testing.T) {
			b, err := json.Marshal(test.usage)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assert.Equal(t, test.expectedJSON, string(b))
		})
	}
}

func TestJSONUnmarshalMetricUsage(t *testing.T) {
	testSuite := []struct {
		name     string
		jason    string
		expected *MetricUsage
	}{
		{
			name:     "empty",
			jason:    `{}`,
			expected: &MetricUsage{},
		},
		{
			name:  "AlertRules",
			jason: `{"alertRules":[{"prom_link":"","group_name":"","name":"foo","expression":""},{"prom_link":"","group_name":"","name":"bar","expression":""}]}`,
			expected: &MetricUsage{
				AlertRules: NewSet(RuleUsage{Name: "foo"}, RuleUsage{Name: "bar"}),
			},
		},
	}
	for _, test := range testSuite {
		t.Run(test.name, func(t *testing.T) {
			var usage MetricUsage
			err := json.Unmarshal([]byte(test.jason), &usage)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assert.Equal(t, test.expected, &usage)
		})
	}
}
