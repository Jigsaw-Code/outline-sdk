import ngrok from "@ngrok/ngrok";
import httpProxy from "http-proxy";
import http from "node:http";
import path from "node:path";
import { createServer } from "vite";
import minimist from "minimist";

export default function main({ 
  entryDomain = "www.example.com",
  proxyToken,
  navigationPath = "/"
}) {
  return new Promise(async (resolve) => {
    const navigationServer = await createServer({
      root: path.join(process.cwd(), "website_proxy"),
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
        target: `https://${entryDomain}`,
        changeOrigin: true
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

      const navigationUrl = listener.url() + navigationPath;
  
      console.log("Externally accessible URL:", navigationUrl);

      resolve({
        entryDomain: listener.url().replace("https://", ''),
        navigationUrl
      })
    });  
  });
}

const getLocalServerUrl = (server) => {
  const addressResult = server.address();
  return typeof addressResult === "string" ? addressResult : `http://[${addressResult.address}]:${addressResult.port}`;
};


if (import.meta.url.endsWith(process.argv[1])) {
  const args = minimist(process.argv.slice(2));

  if (!args.proxyToken) {
    throw new Error("`--proxyToken` must be set!");
  }
  
  main(args).catch(console.error);
}