import ngrok from "@ngrok/ngrok";
import httpProxy from "http-proxy";
import http from "node:http";
import path from "node:path";
import { createServer } from "vite";
import { randomUUID } from "node:crypto";

const getLocalServerUrl = (server) => {
  const addressResult = server.address();
  return typeof addressResult === "string" ? addressResult : `http://[${addressResult.address}]:${addressResult.port}`;
};

export default function main({ 
  mainDomain = process.env.TARGET_DOMAIN || "www.example.com",
  authtoken = process.env.NGROK_TOKEN,
  navigationPath = `/${randomUUID()}`
}) {
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
      if (request.url.startsWith(navigationPath)) {
        return proxyMiddleware.web(request, response, {
          target: getLocalServerUrl(navigationServer.httpServer),
        });
      }
  
      return proxyMiddleware.web(request, response, {
        target: `https://${mainDomain}`,
        changeOrigin: true
      });
    });
  
    proxyServer.listen(async () => {
      const listener = await ngrok.forward({
        addr: getLocalServerUrl(proxyServer),
        authtoken_from_env: false,
        authtoken,
        verify_upstream_tls: false,
        response_header_add: [
          "Allow-Access-Control-Origin: *",
          "Access-Control-Allow-Headers: *",
        ],
      });

      const navigationUrl = listener.url() + navigationPath;
  
      console.log("Externally accessible navigation URL:", navigationUrl);

      resolve({
        mainDomain: listener.url().replace(/$https?\:\/\//, ''),
        navigationUrl
      })
    });  
  });
}

if (import.meta.url.endsWith(process.argv[1])) {
  if (!process.env.NGROK_TOKEN) {
    throw new Error("NGROK_TOKEN must be set!");
  }
  
  main().catch(console.error);
}