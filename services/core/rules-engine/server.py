import http.server
import json
import os
import signal
import sys

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

    def log_message(self, fmt, *args):
        pass


def main():
    port = int(os.getenv("PORT", "8000"))
    server = http.server.ThreadingHTTPServer(("0.0.0.0", port), _Handler)

    def shutdown(signum, frame):
        print("stub: shutting down...", flush=True)
        server.shutdown()
        sys.exit(0)

    signal.signal(signal.SIGINT, shutdown)
    signal.signal(signal.SIGTERM, shutdown)

    print(f"stub listening on :{port}", flush=True)
    server.serve_forever()


if __name__ == "__main__":
    main()
