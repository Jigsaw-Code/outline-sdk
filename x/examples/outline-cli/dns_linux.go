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

func setSystemDNSServer(serverHost string) error {
	setting := []byte(`# Outline CLI DNS Setting
# The original file has been renamed as resolv[.head].outlinecli.backup
nameserver ` + serverHost + "\n")

	if err := backupAndWriteFile(resolvConfFile, resolvConfBackupFile, setting); err != nil {
		return err
	}
	return backupAndWriteFile(resolvConfHeadFile, resolvConfHeadBackupFile, setting)
}

func restoreSystemDNSServer() {
	restoreFileIfExists(resolvConfBackupFile, resolvConfFile)
	restoreFileIfExists(resolvConfHeadBackupFile, resolvConfHeadFile)
}

func backupAndWriteFile(original, backup string, data []byte) error {
	if err := os.Rename(original, backup); err != nil {
		return fmt.Errorf("failed to backup DNS config file '%s' to '%s': %w", original, backup, err)
	}
	if err := os.WriteFile(original, data, 0644); err != nil {
		return fmt.Errorf("failed to write DNS config file '%s': %w", original, err)
	}
	return nil
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
