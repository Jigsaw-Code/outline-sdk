// Copyright 2023 The Outline Authors
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

package shared_backend

import (
	"encoding/json"
	"fmt"
)

// TODO: generalize via reflection/generics and move to infrastructure
type Request struct {
	ResourceName string `json:"resourceName"`
	Parameters   string `json:"parameters"`
}

type Response struct {
	Body  string `json:"body"`
	Error string `json:"error"`
}

func HandleRequest(rawRequest []byte) []byte {
	var request Request

	unmarshallRequestError := json.Unmarshal(rawRequest, &request)

	var response Response

	if unmarshallRequestError != nil {
		response.Error = "HandleRequest: error parsing raw input string"
	}

	var result interface{}
	var resultError error

	if request.ResourceName == "ConnectivityTest" {
		var parameters ConnectivityTestRequest

		unmarshallingParametersError := json.Unmarshal([]byte(request.Parameters), &parameters)

		if unmarshallingParametersError != nil {
			response.Error = "HandleRequest: error parsing method input"
		}

		result, resultError = ConnectivityTest(parameters)
	} else if request.ResourceName == "Platform" {
		result = Platform()
	} else {
		response.Error = "HandleRequest: method name not found"
	}

	if resultError != nil {
		response.Error = resultError.Error()
	}

	rawBody, marshallingBodyError := json.Marshal(result)

	if marshallingBodyError != nil {
		response.Error = "HandleRequest: error serializing method result"
	}

	response.Body = string(rawBody)

	rawResponse, marshallingResponseError := json.Marshal(response)

	if marshallingResponseError != nil {
		fmt.Println("[ERROR] failed to properly marshal HandleRequest output")
	}

	return rawResponse
}
