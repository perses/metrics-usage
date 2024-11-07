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

package labels

import (
	"net/http"

	"github.com/labstack/echo/v4"
	persesEcho "github.com/perses/common/echo"
	"github.com/perses/metrics-usage/database"
)

func NewAPI(db database.Database) persesEcho.Register {
	return &endpoint{
		db: db,
	}
}

type endpoint struct {
	db database.Database
}

func (e *endpoint) RegisterRoute(ech *echo.Echo) {
	path := "/api/v1/labels"
	ech.POST(path, e.PushLabels)
}

func (e *endpoint) PushLabels(ctx echo.Context) error {
	data := make(map[string][]string)
	if err := ctx.Bind(&data); err != nil {
		return ctx.JSON(http.StatusBadRequest, echo.Map{"message": err.Error()})
	}
	if len(data) > 0 {
		e.db.EnqueueLabels(data)
	}
	return ctx.JSON(http.StatusAccepted, echo.Map{"message": "OK"})
}
