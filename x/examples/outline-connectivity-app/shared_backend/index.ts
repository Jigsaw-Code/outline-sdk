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

import { ConnectivityTestInput, ConnectivityTestResult } from "./generated/types"

export function from(backend: Invokable) {
  return {
    connectivityTest: invocationFactory<ConnectivityTestInput, ConnectivityTestResult[]>(backend, "ConnectivityTest")
  }
}

/* INFRASTRUCTURE (to be moved) */

export interface Invokable {
  Invoke(parameters: { method: string; input: string; }): Promise<{ result: string; errors: string[] }>
}

function invocationFactory<T, K>({ Invoke }: Invokable, methodName: string) {
  return async (input: T): Promise<K> => {
    const { result, errors } = await Invoke({ method: methodName, input: JSON.stringify(input) });

    if (errors.length) {
      throw new Error(`from [${methodName}] invocation: ${errors.join(', ')}`);
    }

    return JSON.parse(result) as K;
  }
}
