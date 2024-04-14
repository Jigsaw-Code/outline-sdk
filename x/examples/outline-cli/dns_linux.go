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
	"errors"
	"fmt"
	"os"
)

// todo: find a more portable way of configuring DNS (e.g. resolved)
const (
	resolvConfFile           = "/etc/resolv.conf"
	resolvConfHeadFile       = "/etc/resolv.conf.head"
	resolvConfBackupFile     = "/etc/resolv.outlinecli.backup"
	resolvConfHeadBackupFile = "/etc/resolv.head.outlinecli.backup"
)

// restoreBackup restore backup file function
type restoreBackup func()

func setSystemDNSServer(serverHost string) (restoreBackup, error) {
	setting := []byte(`# Outline CLI DNS Setting
# The original file has been renamed as resolv[.head].outlinecli.backup
nameserver ` + serverHost + "\n")

	restores := make([]restoreBackup, 0, 2)
	restore := func() {
		for _, restore := range restores {
			restore()
		}
	}

	restoreOriginal, err := backupAndWriteFile(resolvConfFile, resolvConfBackupFile, setting)
	restores = append(restores, restoreOriginal)
	if err != nil {
		return restore, err
	}

	restoreOriginalHead, err := backupAndWriteFile(resolvConfHeadFile, resolvConfHeadBackupFile, setting)
	restores = append(restores, restoreOriginalHead)

	return restore, err
}

func backupAndWriteFile(original, backup string, data []byte) (restoreBackup, error) {
	restore := func() {
		// by default restore original file from backup
		restoreFileIfExists(backup, original)
	}

	if err := os.Rename(original, backup); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// original file does not exist, so just remove created file by ourselves
			restore = func() {
				if err := os.Remove(original); err != nil {
					logging.Warn.Printf("failed to remove our file '%s': %v\n", original, err)
				}
			}
		} else {
			return func() {}, fmt.Errorf("failed to backup DNS config file '%s' to '%s': %w", original, backup, err)
		}
	}

	err := os.WriteFile(original, data, 0o644)
	if err != nil {
		return func() {}, fmt.Errorf("failed to write DNS config file '%s': %w", original, err)
	}

	return restore, err
}

func restoreFileIfExists(backup, original string) {
	if _, err := os.Stat(backup); err != nil {
		logging.Warn.Printf("no DNS config backup file '%s' presents: %v\n", backup, err)
		return
	}
	if err := os.Rename(backup, original); err != nil {
		logging.Err.Printf("failed to restore DNS config from backup '%s' to '%s': %v\n", backup, original, err)
		return
	}
	logging.Info.Printf("DNS config restored from '%s' to '%s'\n", backup, original)
}
