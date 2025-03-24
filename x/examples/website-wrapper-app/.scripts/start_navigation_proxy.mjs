import ngrok from "@ngrok/ngrok";
import httpProxy from "http-proxy";
import http from "node:http";
import chalk from "chalk";
import { createServer } from "vite";
import minimist from "minimist";
import { NAVIGATION_PROXY_DIR } from "./lib/constants.mjs";

export default function main({
  entryDomain = "www.example.com",
  proxyToken,
  proxyNavigationPath = "/",
}) {
  return new Promise(async (resolve) => {
    console.log(NAVIGATION_PROXY_DIR);

    const navigationServer = await createServer({
      root: NAVIGATION_PROXY_DIR,
      server: {
        allowedHosts: true,
      },
    });

    await navigationServer.listen();

    const proxyMiddleware = httpProxy.createProxyServer({});
    const proxyServer = http.createServer((request, response) => {
      if (request.url.startsWith(proxyNavigationPath)) {
        return proxyMiddleware.web(request, response, {
          target: getLocalServerUrl(navigationServer.httpServer),
        });
      }

      return proxyMiddleware.web(request, response, {
        target: `https://${entryDomain}`,
        changeOrigin: true,
      });
    });

    proxyServer.listen(async () => {
      const listener = await ngrok.forward({
        addr: getLocalServerUrl(proxyServer),
        authtoken_from_env: false,
        authtoken: proxyToken,
        verify_upstream_tls: false,
        response_header_add: [
          "Allow-Access-Control-Origin: *",
          "Access-Control-Allow-Headers: *",
        ],
      });

      const navigationUrl = listener.url() + proxyNavigationPath;

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

  if (!args.proxyToken) {
    throw new Error("`--proxyToken` must be set!");
  }

  if (!args.proxyNavigationPath) {
    throw new Error("`--proxyNavigationPath` must be set!");
  }

  main(args).catch(console.error);
}
