// Copyright 2024 Jigsaw Operations LLC
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

package smart

import (
	"context"
	"errors"
	"time"
)

// Returns a read channel that is already closed.
func newClosedChanel() <-chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}

// raceTests will call the test function on each entry until it finds an entry for which the test returns nil error.
// That entry is returned. A test is only started after the previous test finished or maxWait is done, whichever
// happens first. That way you bound the wait for a test, and they may overlap.
// The test function should make use of the context to stop doing work when the race is done and it is no longer needed.
func raceTests[E any, R any](ctx context.Context, maxWait time.Duration, entries []E, test func(entry E) (R, error)) (R, error) {
	type testResult struct {
		Result R
		Err    error
	}
	// Communicates the result of each test.
	resultChan := make(chan testResult, len(entries))
	waitCh := newClosedChanel()

	next := 0
	for toTest := len(entries); toTest > 0; {
		select {
		// Search cancelled, quit.
		case <-ctx.Done():
			var empty R
			return empty, ctx.Err()

		// Ready to start testing another resolver.
		case <-waitCh:
			entry := entries[next]
			next++

			waitCtx, waitDone := context.WithTimeout(ctx, maxWait)
			if next == len(entries) {
				// Done with entries. No longer trigger on waitCh.
				waitCh = nil
			} else {
				waitCh = waitCtx.Done()
			}

			go func(entry E, testDone context.CancelFunc) {
				defer testDone()
				result, err := test(entry)
				resultChan <- testResult{Result: result, Err: err}
			}(entry, waitDone)

		// Got a test result.
		case result := <-resultChan:
			toTest--
			if result.Err != nil {
				continue
			}
			return result.Result, nil
		}
	}
	var empty R
	return empty, errors.New("all tests failed")
}
