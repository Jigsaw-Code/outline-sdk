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

package main

/*
// These Cgo directives are essential for compiling on an Apple platform.
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation

#import <Foundation/Foundation.h>
#include <stdlib.h> // For malloc, free, strdup

// A C struct to hold all the process information we want to retrieve.
// This allows us to get all the data in a single Cgo call.
typedef struct {
    char* processName;
    int processIdentifier;
    char* globallyUniqueString;
    char* operatingSystemVersionString;
    char* hostName;
    unsigned long long physicalMemory;
    double systemUptime;
    int processorCount;
    int activeProcessorCount;
    // New fields from your request
    int isMacCatalystApp;
    int isiOSAppOnMac;
    char* userName;
    char* fullUserName;
} ProcessInfo_t;

// A helper function to safely duplicate a C string that might be NULL.
// strdup(NULL) is undefined behavior, so this prevents crashes.
static char* safe_strdup(const char* s) {
    if (s == NULL) {
        // Return a dynamically allocated empty string.
        return strdup("");
    }
    return strdup(s);
}


// This C function (using Objective-C) populates and returns a struct
// containing a wide range of process information.
static ProcessInfo_t* get_all_process_info() {
    // Autorelease pool is good practice for managing memory in Objective-C code
    // called from other languages.
    @autoreleasepool {
        NSProcessInfo *info = [NSProcessInfo processInfo];

        // Allocate memory for our C struct.
        ProcessInfo_t *p_info = (ProcessInfo_t*)malloc(sizeof(ProcessInfo_t));
        if (p_info == NULL) {
            return NULL; // Failed to allocate memory
        }

        // Use safe_strdup to create heap-allocated C string copies that Go can safely manage and free.
        p_info->processName = safe_strdup([[info processName] UTF8String]);
        p_info->globallyUniqueString = safe_strdup([[info globallyUniqueString] UTF8String]);
        p_info->operatingSystemVersionString = safe_strdup([[info operatingSystemVersionString] UTF8String]);
        p_info->hostName = safe_strdup([[info hostName] UTF8String]);
        // p_info->userName = safe_strdup([[info userName] UTF8String]);
        // p_info->fullUserName = safe_strdup([[info fullUserName] UTF8String]);

        // Populate numeric fields directly.
        p_info->processIdentifier = [info processIdentifier];
        p_info->physicalMemory = [info physicalMemory];
        p_info->systemUptime = [info systemUptime];
        p_info->processorCount = (int)[info processorCount];
        p_info->activeProcessorCount = (int)[info activeProcessorCount];

        // Populate boolean flags (as integers), checking for API availability.
        if (@available(macOS 10.15, iOS 13.0, *)) {
            p_info->isMacCatalystApp = [info isMacCatalystApp] ? 1 : 0;
        } else {
            p_info->isMacCatalystApp = 0; // Default to false on older systems.
        }

        if (@available(macOS 11.0, iOS 14.0, *)) {
            p_info->isiOSAppOnMac = [info isiOSAppOnMac] ? 1 : 0;
        } else {
            p_info->isiOSAppOnMac = 0; // Default to false on older systems.
        }

        return p_info;
    }
}
*/
import "C"
import (
	"fmt"
	"log"
	"unsafe"
)

// A Go struct that mirrors the C struct, providing an idiomatic way
// to work with the process information in Go.
type ProcessInfo struct {
	ProcessName                  string
	ProcessIdentifier            int
	GloballyUniqueString         string
	OperatingSystemVersionString string
	HostName                     string
	PhysicalMemoryBytes          uint64
	SystemUptimeSeconds          float64
	ProcessorCount               int
	ActiveProcessorCount         int
	IsMacCatalystApp             bool
	IsIOSAppOnMac                bool
	UserName                     string
	FullUserName                 string
}

// getProcessInfo is a Go wrapper function that calls the underlying C function
// and converts the C struct into a Go struct.
func getProcessInfo() (*ProcessInfo, error) {
	// Call the C function to get the populated struct.
	cInfo := C.get_all_process_info()

	// Check if the C function returned NULL, which indicates an error.
	if cInfo == nil {
		return nil, fmt.Errorf("failed to get process info from NSProcessInfo")
	}

	// The memory for the C struct and its string members was allocated in C.
	// We must free all of it to prevent memory leaks. The defer statements
	// ensure C.free is called for each allocated piece of memory right
	// before the function returns.
	defer C.free(unsafe.Pointer(cInfo.processName))
	defer C.free(unsafe.Pointer(cInfo.globallyUniqueString))
	defer C.free(unsafe.Pointer(cInfo.operatingSystemVersionString))
	defer C.free(unsafe.Pointer(cInfo.hostName))
	defer C.free(unsafe.Pointer(cInfo.userName))
	defer C.free(unsafe.Pointer(cInfo.fullUserName))
	defer C.free(unsafe.Pointer(cInfo))

	// Create a Go struct and copy the data from the C struct, converting types as needed.
	goInfo := &ProcessInfo{
		ProcessName:                  C.GoString(cInfo.processName),
		ProcessIdentifier:            int(cInfo.processIdentifier),
		GloballyUniqueString:         C.GoString(cInfo.globallyUniqueString),
		OperatingSystemVersionString: C.GoString(cInfo.operatingSystemVersionString),
		HostName:                     C.GoString(cInfo.hostName),
		PhysicalMemoryBytes:          uint64(cInfo.physicalMemory),
		SystemUptimeSeconds:          float64(cInfo.systemUptime),
		ProcessorCount:               int(cInfo.processorCount),
		ActiveProcessorCount:         int(cInfo.activeProcessorCount),
		IsMacCatalystApp:             cInfo.isMacCatalystApp != 0,
		IsIOSAppOnMac:                cInfo.isiOSAppOnMac != 0,
		UserName:                     C.GoString(cInfo.userName),
		FullUserName:                 C.GoString(cInfo.fullUserName),
	}

	return goInfo, nil
}

func main() {
	fmt.Println("Attempting to get iOS process info using Cgo...")

	// Call our Go wrapper function.
	info, err := getProcessInfo()
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Print all the retrieved information in a formatted way.
	fmt.Printf("\n--- Successfully Retrieved Process Info ---\n")
	fmt.Printf("Process Name:           %s\n", info.ProcessName)
	fmt.Printf("Process ID (PID):       %d\n", info.ProcessIdentifier)
	fmt.Printf("User Name:              %s\n", info.UserName)
	fmt.Printf("Full User Name:         %s\n", info.FullUserName)
	fmt.Printf("Globally Unique ID:     %s\n", info.GloballyUniqueString)
	fmt.Printf("OS Version:             %s\n", info.OperatingSystemVersionString)
	fmt.Printf("Hostname:               %s\n", info.HostName)
	fmt.Printf("Is Mac Catalyst App:    %t\n", info.IsMacCatalystApp)
	fmt.Printf("Is iOS App on Mac:      %t\n", info.IsIOSAppOnMac)
	fmt.Printf("Physical Memory (B):    %d\n", info.PhysicalMemoryBytes)
	fmt.Printf("System Uptime (s):      %.2f\n", info.SystemUptimeSeconds)
	fmt.Printf("Processor Count:        %d\n", info.ProcessorCount)
	fmt.Printf("Active Processor Count: %d\n", info.ActiveProcessorCount)
	fmt.Println("-------------------------------------------")
}
