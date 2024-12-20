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

type systemDNSBackup struct {
	original, backup string
}

var systemDNSBackups = make([]systemDNSBackup, 0, 2)

func setSystemDNSServer(serverHost string) error {
	setting := []byte(`# Outline CLI DNS Setting
# The original file has been renamed as resolv[.head].outlinecli.backup
nameserver ` + serverHost + "\n")

	err := backupAndWriteFile(resolvConfFile, resolvConfBackupFile, setting)
	if err != nil {
		return err
	}

	err = backupAndWriteFile(resolvConfHeadFile, resolvConfHeadBackupFile, setting)
	if err != nil {
		return err
	}

	return nil
}

func backupAndWriteFile(original, backup string, data []byte) error {
	if _, err := os.Stat(original); err == nil {
		// original file exist - move it into backup
		if err := os.Rename(original, backup); err != nil {
			return fmt.Errorf("failed to backup DNS config file '%s' to '%s': %w", original, backup, err)
		}
	} else if !errors.Is(err, os.ErrNotExist) { // if not exist - it's ok, just write to it
		return fmt.Errorf("failed to check the existence of DNS config file '%s': %w", original, err)
	}

	systemDNSBackups = append(systemDNSBackups, systemDNSBackup{
		original: original,
		backup:   backup,
	})

	// save data to original
	if err := os.WriteFile(original, data, 0o644); err != nil {
		return fmt.Errorf("failed to write DNS config file '%s': %w", original, err)
	}

	return nil
}

func restoreSystemDNSServer() {
	for _, backup := range systemDNSBackups {
		if _, err := os.Stat(backup.backup); err == nil {
			// backup exist - replace original with it
			if err := os.Rename(backup.backup, backup.original); err != nil {
				logging.Err.Printf(
					"failed to restore DNS config from backup '%s' to '%s': %v\n",
					backup.backup,
					backup.original,
					err,
				)
				continue
			}
			logging.Info.Printf("DNS config restored from '%s' to '%s'\n", backup.backup, backup.original)
		} else if errors.Is(err, os.ErrNotExist) {
			// backup not exist - just remove original, because it's created by ourselves
			if err := os.Remove(backup.original); err != nil {
				logging.Err.Printf("failed to remove Outline DNS config file '%s': %v\n", backup.original, err)
				continue
			}
			logging.Info.Printf("Outline DNS config '%s' has been removed\n", backup.original)
		} else {
			logging.Err.Printf("failed to check the existence of DNS config backup file '%s': %v\n", backup.backup, err)
		}
	}
}
