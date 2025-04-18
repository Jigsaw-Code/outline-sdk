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

import ngrok from "@ngrok/ngrok";
import httpProxy from "http-proxy";
import http from "node:http";
import chalk from "chalk";
import path from "node:path";
import { createServer } from "vite";
import minimist from "minimist";

export default function main({
  entryUrl = "https://www.example.com",
  navigatorToken,
  navigatorPath = "/nav",
}) {
  return new Promise(async (resolve) => {
    const navigationServer = await createServer({
      root: path.join(process.cwd(), "basic_navigator_example/src"),
      server: {
        allowedHosts: true,
      },
    });

    await navigationServer.listen();

    const proxyMiddleware = httpProxy.createProxyServer({});
    const proxyServer = http.createServer((request, response) => {
      if (request.url.startsWith(navigatorPath)) {
        return proxyMiddleware.web(request, response, {
          target: getLocalServerUrl(navigationServer.httpServer),
        });
      }

      return proxyMiddleware.web(request, response, {
        target: entryUrl,
        changeOrigin: true,
      });
    });

    proxyServer.listen(async () => {
      const listener = await ngrok.forward({
        addr: getLocalServerUrl(proxyServer),
        authtoken_from_env: false,
        authtoken: navigatorToken,
        verify_upstream_tls: false,
        response_header_add: [
          "Allow-Access-Control-Origin: *",
          "Access-Control-Allow-Headers: *",
        ],
      });

      const navigationUrl = listener.url() + navigatorPath;

      console.log("Navigation proxy running @", chalk.blue(navigationUrl));

      resolve({
        entryDomain: new URL(listener.url()).host,
        navigationUrl,
      });
    });
  });
}

const getLocalServerUrl = (server) => {
  const addressResult = server.address();
  return typeof addressResult === "string"
    ? addressResult
    : `http://[${addressResult.address}]:${addressResult.port}`;
};

if (import.meta.url.endsWith(process.argv[1])) {
  const args = minimist(process.argv.slice(2));

  if (!args.navigatorToken) {
    throw new Error("`--navigatorToken` must be set!");
  }

  main(args).catch(console.error);
}
