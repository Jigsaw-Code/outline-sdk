import ngrok from "@ngrok/ngrok";
import handler from "serve-handler";
import http from "http";

const PORT = process.env.PORT || 3000;

if (!process.env.NGROK_TOKEN || !process.env.NGROK_DOMAIN) {
  throw new Error("NGROK_TOKEN and NGROK_DOMAIN must be set!");
}

const server = http.createServer((request, response) => {
  return handler(request, response, {
    public: "www",
  });
});

server.listen(PORT, async () => {
  const listener = await ngrok.forward({
    domain: process.env.NGROK_DOMAIN,
    addr: `http://127.0.0.1:${PORT}`,
    authtoken_from_env: false,
    authtoken: process.env.NGROK_TOKEN,
    response_header_add: [
      "Allow-Access-Control-Origin: *",
      "Access-Control-Allow-Headers: *",
    ],
  });

  console.log(`Listening on '${listener.url()}'`);
});
