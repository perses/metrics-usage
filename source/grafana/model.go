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

type target struct {
	Expr string `json:"expr,omitempty"`
}

type panel struct {
	Type    string   `json:"type"`
	Title   string   `json:"title"`
	Panels  []panel  `json:"panels"`
	Targets []target `json:"targets"`
}

type row struct {
	Panels []panel `json:"panels"`
}

type templateVar struct {
	Name  string      `json:"name"`
	Type  string      `json:"type"`
	Query interface{} `json:"query"`
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
	if queryObj, ok := v.Query.(map[string]interface{}); ok {
		if query, ok := queryObj["query"].(string); ok {
			return query, nil
		}
	}
	return "", fmt.Errorf("query variable %q doesn't have the right type (string, or JSON object). It is of type %T", v.Name, v.Query)
}

type simplifiedDashboard struct {
	UID        string  `json:"uid,omitempty"`
	Title      string  `json:"title"`
	Panels     []panel `json:"panels"`
	Rows       []row   `json:"rows"`
	Templating struct {
		List []templateVar `json:"list"`
	} `json:"templating"`
}

func extractTarget(panel panel) []target {
	var targets []target
	for _, p := range panel.Panels {
		targets = append(targets, extractTarget(p)...)
	}
	return append(targets, panel.Targets...)
}
