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

package utils

func InsertIfNotPresent[T comparable](slice []T, item T) []T {
	for _, s := range slice {
		if item == s {
			return slice
		}
	}
	return append(slice, item)
}

func Merge[T comparable](old, new []T) []T {
	if len(old) == 0 {
		return new
	}
	if len(new) == 0 {
		return old
	}
	for _, a := range old {
		found := false
		for _, b := range new {
			if a == b {
				found = true
				break
			}
		}
		if !found {
			new = append(new, a)
		}
	}
	return new
}
