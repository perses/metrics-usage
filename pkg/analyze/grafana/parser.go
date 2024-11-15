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

func extractMetricNameWithVariable(expr string) []string {
	p := &parser{}
	return p.parse(expr)
}

type parser struct {
	metrics       []string
	currentMetric string
}

func (p *parser) parse(expr string) []string {
	query := []rune(expr)
	for i := 0; i < len(query); i++ {
		char := query[i]
		if isWhitespace(char) {
			// Whitespace
			continue
		}
		if isValidMetricChar(char) {
			p.currentMetric += string(char)
			continue
		}
		if char == '(' || char == ')' || char == '"' || char == '=' || char == '!' || char == ',' {
			// then it was not a metric name and we need to drop it
			p.currentMetric = ""
			continue
		}
		if char == '$' {
			// That means we are starting to collect a variable hopefully into a metric name.
			p.currentMetric += string(char)
			if i+1 < len(query) && query[i+1] == '{' {
				// if the variable is between bracket, we should loop other it to ensure we don't miss it by stopping collecting the metric
				for j := i + 1; query[j] != '}' && j < len(query); j++ {
					i = j
					if isWhitespace(query[j]) {
						continue
					}
					p.currentMetric += string(query[j])
				}
				// here we increment i to match the value of j when it get out of the loop above.
				i++
				if i < len(query) {
					// At this moment, query[i] == '}'
					p.currentMetric += string(query[i])
				}
			}
			continue
		}
		if char == '{' {
			if len(p.currentMetric) > 0 {
				// That means we reached the end of a metric, so we can save it
				p.metrics = append(p.metrics, p.currentMetric)
				p.currentMetric = ""
			}
		}
	}
	return p.metrics
}

func isWhitespace(ch rune) bool {
	return ch == ' ' || ch == '\t' || ch == '\n'
}

func isValidMetricChar(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' || ch == ':'
}
