"use strict";
const http = require("http");
// M0 stub — replaced by the real implementation in its milestone (ROADMAP.md).
const port = parseInt(process.env.PORT, 10) || 3000;
const server = http.createServer((req, res) => {
  const payload = req.url === "/healthz"
    ? { status: "ok" }
    : { service: process.env.OTEL_SERVICE_NAME || "portal-stub" };
  res.writeHead(200, { "Content-Type": "application/json" });
  res.end(JSON.stringify(payload));
});
server.listen(port, () => console.log("stub listening on :" + port));

// Graceful shutdown
const shutdown = () => { console.log("stub: shutting down..."); server.close(() => process.exit(0)); };
process.on("SIGINT", shutdown);
process.on("SIGTERM", shutdown);
