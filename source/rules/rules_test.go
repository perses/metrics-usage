// Copyright The Perses Authors
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

package rules

import (
	"context"
	"errors"
	"testing"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

type fakePromAPI struct {
	v1.API
	err error
	// onRules is called each time the rules are requested.
	onRules func()
}

func (f fakePromAPI) Rules(_ context.Context, _ []string) (v1.RulesResult, error) {
	f.onRules()
	return v1.RulesResult{}, f.err
}

func TestExecute_ReturnErrorWhenGettingRulesFails(t *testing.T) {
	rulesErr := errors.New("prometheus unavailable")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := &rulesCollector{
		// The context is canceled when the rules are requested, so the collector doesn't wait before giving up.
		promClient: fakePromAPI{err: rulesErr, onRules: cancel},
		retry:      1,
		logger:     logrus.NewEntry(logrus.StandardLogger()),
	}
	assert.ErrorIs(t, c.Execute(ctx, cancel), rulesErr)
}
