// Copyright 2023 Jigsaw Operations LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"encoding/json"

	"github.com/Jigsaw-Code/outline-sdk/x/examples/outline-connectivity-app/shared_backend"
)

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) Request(resourceName string, parameters interface{}) interface{} {
	rawRequest, marshallingError := json.Marshal(parameters)

	if marshallingError != nil {
		return nil, "Request: failed to serialize request parameters"
	}

	var response shared_backend.Response

	unmarshallingError := json.Unmarshal(shared_backend.HandleRequest(rawRequest), &response)

	if unmarshallingError != nil {
		return nil, "Request: failed to parse request result"
	}

	return response
}
