// Container healthcheck script.
// Uses the built-in http module so it works on node:20-slim without wget/curl.

const http = require("http");

const req = http.get("http://localhost:3000/api/healthz", (res) => {
  process.exit(res.statusCode === 200 ? 0 : 1);
});

req.on("error", () => process.exit(1));
req.setTimeout(5000, () => {
  req.destroy();
  process.exit(1);
});
