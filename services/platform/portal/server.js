"use strict";
const http = require("http");
// M0 stub — replaced by the real implementation in its milestone (ROADMAP.md).
const port = 3000;
const server = http.createServer(function (req, res) {
  if (req.url === "/healthz") {
    res.writeHead(200, { "Content-Type": "application/json" });
    res.end("{\"status\":\"ok\"}");
  } else {
    res.writeHead(200, { "Content-Type": "application/json" });
    res.end("{\"service\":\"" + (process.env.OTEL_SERVICE_NAME || "portal-stub") + "\"}");
  }
});
server.listen(port, function () { console.log("stub listening on :" + port); });
