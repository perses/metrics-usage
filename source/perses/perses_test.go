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

package perses

import (
	"context"
	"errors"
	"testing"

	persesClientV1 "github.com/perses/perses/pkg/client/api/v1"
	v1 "github.com/perses/perses/pkg/model/api/v1"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

type fakeDashboardClient struct {
	persesClientV1.DashboardInterface
	err error
}

func (f fakeDashboardClient) List(_ string) ([]*v1.Dashboard, error) {
	return nil, f.err
}

func TestExecute_ReturnErrorWhenListingDashboardsFails(t *testing.T) {
	listErr := errors.New("perses unavailable")
	c := &persesCollector{
		persesClient: fakeDashboardClient{err: listErr},
		logger:       logrus.NewEntry(logrus.StandardLogger()),
	}
	assert.ErrorIs(t, c.Execute(context.Background(), func() {}), listErr)
}
