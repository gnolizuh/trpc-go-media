//
// Copyright [2024] [https://github.com/gnolizuh]
//
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
//

// Package frame contains transport-level frame utilities.
package frame

// ShouldCopy judges whether to enable frame copy according to the current framer and options.
func ShouldCopy(isCopyOption, serverAsync, isSafeFramer bool) bool {
	// The following two scenarios do not need to copy frame.
	// Scenario 1: Framer is already safe on concurrent read.
	if isSafeFramer {
		return false
	}
	// Scenario 2: The server is in sync mod, and the caller does not want to copy frame(not stream RPC).
	if !serverAsync && !isCopyOption {
		return false
	}

	// Otherwise:
	// To avoid data overwriting of the concurrent reading Framer, enable copy frame by default.
	return true
}
