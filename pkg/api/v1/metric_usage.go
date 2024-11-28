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

package v1

import (
	"encoding/json"

	"github.com/perses/perses/pkg/model/api/v1/common"
)

type Set[T comparable] map[T]struct{}

func NewSet[T comparable](vals ...T) Set[T] {
	s := Set[T]{}
	for _, v := range vals {
		s[v] = struct{}{}
	}
	return s
}

func MergeSet[T comparable](old, new Set[T]) Set[T] {
	if new == nil {
		return old
	}
	if old == nil {
		return new
	}
	s := Set[T]{}
	for k, v := range new {
		s[k] = v
	}
	for k, v := range old {
		s[k] = v
	}
	return s
}

func (s Set[T]) Add(vals ...T) {
	for _, v := range vals {
		s[v] = struct{}{}
	}
}

func (s Set[T]) Remove(value T) {
	delete(s, value)
}

func (s Set[T]) Contains(value T) bool {
	_, ok := s[value]
	return ok
}

func (s Set[T]) Merge(other Set[T]) {
	for v := range other {
		s.Add(v)
	}
}

func (s Set[T]) TransformAsSlice() []T {
	if s == nil {
		return nil
	}
	var slice []T
	for v := range s {
		slice = append(slice, v)
	}
	return slice
}

func (s Set[T]) MarshalJSON() ([]byte, error) {
	if s == nil {
		return []byte("[]"), nil
	}
	var slice []T
	for v := range s {
		slice = append(slice, v)
	}
	return json.Marshal(slice)
}

func (s *Set[T]) UnmarshalJSON(b []byte) error {
	var slice []T
	if err := json.Unmarshal(b, &slice); err != nil {
		return err
	}
	if len(slice) == 0 {
		return nil
	}
	*s = make(map[T]struct{}, len(slice))
	for _, v := range slice {
		s.Add(v)
	}
	return nil
}

type RuleUsage struct {
	PromLink   string `json:"prom_link"`
	GroupName  string `json:"group_name"`
	Name       string `json:"name"`
	Expression string `json:"expression"`
}

type DashboardUsage struct {
	ID   string `json:"uid"`
	Name string `json:"title"`
	URL  string `json:"url"`
}

type MetricUsage struct {
	Dashboards     Set[DashboardUsage] `json:"dashboards,omitempty"`
	RecordingRules Set[RuleUsage]      `json:"recordingRules,omitempty"`
	AlertRules     Set[RuleUsage]      `json:"alertRules,omitempty"`
}

func MergeUsage(old, new *MetricUsage) *MetricUsage {
	if old == nil {
		return new
	}
	if new == nil {
		return old
	}
	return &MetricUsage{
		Dashboards:     MergeSet(old.Dashboards, new.Dashboards),
		AlertRules:     MergeSet(old.AlertRules, new.AlertRules),
		RecordingRules: MergeSet(old.RecordingRules, new.RecordingRules),
	}
}

type Metric struct {
	Labels Set[string]  `json:"labels,omitempty"`
	Usage  *MetricUsage `json:"usage,omitempty"`
}

type PartialMetric struct {
	Usage           *MetricUsage   `json:"usage,omitempty"`
	MatchingMetrics Set[string]    `json:"matchingMetrics,omitempty"`
	MatchingRegexp  *common.Regexp `json:"matchingRegexp,omitempty"`
}
