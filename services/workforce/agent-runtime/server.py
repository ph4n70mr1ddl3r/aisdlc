import http.server
import json
import os

# M0 stub — replaced by the real implementation in its milestone (ROADMAP.md).


class _Handler(http.server.BaseHTTPRequestHandler):
    def _send(self, code, payload):
        body = json.dumps(payload).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_GET(self):
        if self.path == "/healthz":
            self._send(200, {"status": "ok"})
        else:
            self._send(200, {"service": os.getenv("OTEL_SERVICE_NAME", "stub")})

    def log_message(self, fmt, *args):  # quieter logs
        return


if __name__ == "__main__":
    port = int(os.getenv("PORT", "8000"))
    print("stub listening on :" + str(port), flush=True)
    http.server.ThreadingHTTPServer(("0.0.0.0", port), _Handler).serve_forever()
