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

import "fmt"

type Target struct {
	Expr string `json:"expr,omitempty"`
}

type Panel struct {
	Type    string   `json:"type"`
	Title   string   `json:"title"`
	Panels  []Panel  `json:"panels"`
	Targets []Target `json:"targets"`
}

type row struct {
	Panels []Panel `json:"panels"`
}

type option struct {
	Value string `json:"value"`
}

type templateVar struct {
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	Query   any      `json:"query"`
	Options []option `json:"options"`
}

// extractQueryFromVariableTemplating will extract the PromQL expression from query.
// Query can have two types.
// It can be a string or the following JSON object:
// { query: "up", refId: "foo" }
// We need to ensure we are in one of the different cases.
func (v templateVar) extractQueryFromVariableTemplating() (string, error) {
	if query, ok := v.Query.(string); ok {
		return query, nil
	}
	if queryObj, ok := v.Query.(map[string]any); ok {
		if query, ok := queryObj["query"].(string); ok {
			return query, nil
		}
	}
	return "", fmt.Errorf("unable to extract the query expression from the variable %q", v.Name)
}

type SimplifiedDashboard struct {
	UID        string  `json:"uid,omitempty"`
	Title      string  `json:"title"`
	Panels     []Panel `json:"panels"`
	Rows       []row   `json:"rows"`
	Templating struct {
		List []templateVar `json:"list"`
	} `json:"templating"`
}

func extractTarget(panel Panel) []Target {
	var targets []Target
	for _, p := range panel.Panels {
		targets = append(targets, extractTarget(p)...)
	}
	return append(targets, panel.Targets...)
}
