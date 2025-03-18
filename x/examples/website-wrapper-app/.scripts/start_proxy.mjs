import ngrok from "@ngrok/ngrok";
import httpProxy from "http-proxy";
import http from "node:http";
import path from "node:path";
import { createServer } from "vite";
import { randomUUID } from "node:crypto";

const NAVIGATION_PATH = `/${randomUUID()}`;
const TARGET_DOMAIN = process.env.TARGET_DOMAIN || "www.example.com";

if (!process.env.NGROK_TOKEN) {
  throw new Error("NGROK_TOKEN must be set!");
}

const getLocalServerUrl = (server) => {
  const addressResult = server.address();
  return typeof addressResult === "string" ? addressResult : `http://[${addressResult.address}]:${addressResult.port}`;
};

export default function main() {
  return new Promise(async (resolve) => {
    const navigationServer = await createServer({
      root: path.join(process.cwd(), "navigation_proxy"),
      server: {
        allowedHosts: true
      }
    });
  
    await navigationServer.listen();
  
    const proxyMiddleware = httpProxy.createProxyServer({});
    const proxyServer = http.createServer((request, response) => {
      if (request.url.startsWith(NAVIGATION_PATH)) {
        return proxyMiddleware.web(request, response, {
          target: getLocalServerUrl(navigationServer.httpServer),
        });
      }
  
      return proxyMiddleware.web(request, response, {
        target: `https://${TARGET_DOMAIN}`,
        changeOrigin: true
      });
    });
  
    proxyServer.listen(async () => {
      const listener = await ngrok.forward({
        addr: getLocalServerUrl(proxyServer),
        authtoken_from_env: false,
        authtoken: process.env.NGROK_TOKEN,
        verify_upstream_tls: false,
        response_header_add: [
          "Allow-Access-Control-Origin: *",
          "Access-Control-Allow-Headers: *",
        ],
      });
  
      console.log("Externally accessible navigation URL:", listener.url() + NAVIGATION_PATH);

      resolve({
        mainDomain: listener.url().replace('https://', '').replace('http://', ''),
        navigationUrl: listener.url() + NAVIGATION_PATH
      })
    });  
  });
}

if (import.meta.url.endsWith(process.argv[1])) {
  main().catch(console.error);
}