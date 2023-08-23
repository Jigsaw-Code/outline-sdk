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
