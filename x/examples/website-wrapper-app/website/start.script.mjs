import ngrok from "@ngrok/ngrok";
import httpProxy from "http-proxy";
import http from "node:http";
import { fileURLToPath } from "node:url";
import { createServer } from "vite";

const TARGET_DOMAIN = process.env.TARGET_DOMAIN || "www.example.com";
const __dirname = fileURLToPath(new URL(".", import.meta.url));

if (!process.env.NGROK_TOKEN) {
  throw new Error("NGROK_TOKEN and NGROK_DOMAIN must be set!");
}

const getLocalServerUrl = (server) => {
  const addressResult = server.address();
  return typeof addressResult === "string" ? addressResult : `http://[${addressResult.address}]:${addressResult.port}`;
};


(async function () {
  const navigationServer = await createServer({
    root: __dirname
  });

  await navigationServer.listen();

  console.log(`Navigation server: ${getLocalServerUrl(navigationServer.httpServer)}`);

  const proxyMiddleware = httpProxy.createProxyServer({});
  const proxyServer = http.createServer((request, response) => {
    if (request.url.startsWith("/nav")) {
      return proxyMiddleware.web(request, response, {
        target: getLocalServerUrl(navigationServer.httpServer),
      });
    }

    return proxyMiddleware.web(request, response, {
      target: `http://${TARGET_DOMAIN}`,
    });
  });

  proxyServer.listen(async () => {
    console.log(`Proxy server: ${getLocalServerUrl(proxyServer)}`);

    const listener = await ngrok.forward({
      addr: getLocalServerUrl(proxyServer),
      authtoken_from_env: false,
      authtoken: process.env.NGROK_TOKEN,
      response_header_add: [
        "Allow-Access-Control-Origin: *",
        "Access-Control-Allow-Headers: *",
      ],
    });

    console.log(`TLS frontend URL: ${listener.url()}`);
  });
})();
