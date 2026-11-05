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

package labels

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/perses/metrics-usage/pkg/client"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// fakePromAPI returns the metric "up" with the label "job", unless labelValuesErr is set.
type fakePromAPI struct {
	v1.API
	labelValuesErr error
}

func (f fakePromAPI) LabelValues(_ context.Context, _ string, _ []string, _, _ time.Time, _ ...v1.Option) (model.LabelValues, v1.Warnings, v1.Infos, error) {
	if f.labelValuesErr != nil {
		return nil, nil, nil, f.labelValuesErr
	}
	return model.LabelValues{"up"}, nil, nil, nil
}

func (f fakePromAPI) LabelNames(_ context.Context, _ []string, _, _ time.Time, _ ...v1.Option) (model.LabelNames, v1.Warnings, v1.Infos, error) {
	return model.LabelNames{model.MetricNameLabel, "job"}, nil, nil, nil
}

type fakeMetricUsageClient struct {
	client.Client
	err error
}

func (f fakeMetricUsageClient) Labels(_ map[string][]string) error {
	return f.err
}

func newCollector(promClient v1.API, metricUsageClient client.Client) *labelCollector {
	return &labelCollector{
		promClient:        promClient,
		metricUsageClient: metricUsageClient,
		concurrency:       1,
		logger:            logrus.NewEntry(logrus.StandardLogger()),
	}
}

func TestExecute_ReturnErrorWhenQueryFails(t *testing.T) {
	queryErr := errors.New("prometheus unavailable")
	c := newCollector(fakePromAPI{labelValuesErr: queryErr}, fakeMetricUsageClient{})
	assert.ErrorIs(t, c.Execute(context.Background(), func() {}), queryErr)
}

func TestExecute_ReturnErrorWhenSendingLabelsFails(t *testing.T) {
	sendErr := errors.New("metrics usage server unavailable")
	c := newCollector(fakePromAPI{}, fakeMetricUsageClient{err: sendErr})
	assert.ErrorIs(t, c.Execute(context.Background(), func() {}), sendErr)
}
