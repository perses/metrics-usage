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

package metric

import (
	"context"
	"errors"
	"testing"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

type fakePromAPI struct {
	v1.API
	err error
}

func (f fakePromAPI) LabelValues(_ context.Context, _ string, _ []string, _, _ time.Time, _ ...v1.Option) (model.LabelValues, v1.Warnings, v1.Infos, error) {
	return nil, nil, nil, f.err
}

func TestExecute_ReturnErrorWhenQueryFails(t *testing.T) {
	queryErr := errors.New("prometheus unavailable")
	c := &metricCollector{
		client: fakePromAPI{err: queryErr},
		logger: logrus.NewEntry(logrus.StandardLogger()),
	}
	assert.ErrorIs(t, c.Execute(context.Background(), func() {}), queryErr)
}
