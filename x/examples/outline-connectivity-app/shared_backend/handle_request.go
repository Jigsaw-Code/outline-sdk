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

package shared_backend

/* INFRASTRUCTURE (to be generalized via reflection/generics and moved) */
type Request struct {
	Name string `json:"method"`
	Parameters  string `json:"input"`
}

type Response struct {
	Body string   `json:"result"`
	Error string    `json:"error"`
}

func HandleRequest(rawRequest []byte) []byte {
	var request Request

	unmarshallRequestError := json.Unmarshal(rawRequest, &request)

	var response Response

	if unmarshallRequestError != nil {
		response.Error = "HandleIPC: error parsing raw input string";
	}

	/* TODO: generalize, make non-blocking */
	if request.Name != "ConnectivityTest" {
		response.Error = "HandleIPC: method name not found";
	}

	var parameters ConnectivityTestParameters

	unmarshallingParametersError := json.Unmarshal([]byte(request.Parameters), &parameters)

	if unmarshallingParametersError != nil {
		response.Error = "HandleIPC: error parsing method input"
	}

	result, testError := ConnectivityTest(parameters)

	if testError != nil {
		response.Error = testError.Error()
	}

	rawBody, marshallingBodyError := json.Marshal(result)

	if marshallingBodyError != nil {
		response.Error = "HandleIPC: error serializing method result"
	}
  /* END TODO: generalize, make non-blocking */

	response.Body = string(rawBody)

	rawResponse, marshallingResponseError := json.Marshal(response)

	if marshallingResponseError != nil {
		fmt.Println("[ERROR] failed to properly marshal HandleIPC output")
	}

	return rawResponse
}
