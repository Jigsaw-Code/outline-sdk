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

export interface ConnectivityTestRequest {
  accessKey: string;
  domain: string;
  resolvers: string[];
  protocols: {
    tcp: boolean;
    udp: boolean;
  };
}

export interface ConnectivityTestResult {
  time: string;
  durationMs: number;
  proto: string;
  resolver: string;
  error?: {
    operation: string;
    posixError: string;
    message: string;
  };
}

export type ConnectivityTestResponse =  ConnectivityTestResult[] | Error | null;

export enum OperatingSystem {
  ANDROID = "android",
  IOS = "ios",
  LINUX = "linux",
  MACOS = "darwin",
  WINDOWS = "windows"
}

export interface PlatformMetadata {
  operatingSystem: OperatingSystem;
}