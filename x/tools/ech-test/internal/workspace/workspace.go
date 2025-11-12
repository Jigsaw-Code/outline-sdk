// Copyright 2025 The Outline Authors
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

package workspace

import (
	"log/slog"
	"os"
	"path/filepath"
)

// EnsureWorkspace ensures the workspace directory exists, creating it if needed.
func EnsureWorkspace(workspaceDir string) string {
	workspaceAbsDir, err := filepath.Abs(workspaceDir)
	if err != nil {
		slog.Error("Failed to resolve workspace path", "error", err)
		os.Exit(1)
	}
	if _, err := os.Stat(workspaceAbsDir); os.IsNotExist(err) {
		slog.Info("Creating workspace directory", "path", workspaceAbsDir)
		if err := os.MkdirAll(workspaceAbsDir, 0755); err != nil {
			slog.Error("Failed to create workspace directory", "error", err)
			os.Exit(1)
		}
	}
	return workspaceAbsDir
}
