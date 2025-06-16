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

package tlsfrag

// Split SNI implemented FragFunc.

func MakeSplitSniFunc(sniSplit int) FragFunc {
	// takes in an int, and returns a FragFunc which splits the on the sni

	fragFunc := func(clientHello []byte) int {
		return sniSplit
	}

	return fragFunc
}
