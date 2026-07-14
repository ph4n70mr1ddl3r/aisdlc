"""Shared M0 stub HTTP server for Python services.

Each service replaces this with its real implementation per ROADMAP.md
milestones. The service Dockerfiles vendor this file as ``_stub.py`` next
to their ``server.py``, so a service entrypoint is just::

    import _stub
    _stub.run_stub()
"""

import http.server
import json
import os
import signal


class _Handler(http.server.BaseHTTPRequestHandler):
    def _send(self, code, payload):
        body = json.dumps(payload).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.send_header("Access-Control-Allow-Origin", "*")
        self.send_header("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
        self.send_header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Tenant-ID")
        self.end_headers()
        self.wfile.write(body)

    def do_OPTIONS(self):
        self.send_response(204)
        self.send_header("Access-Control-Allow-Origin", "*")
        self.send_header("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
        self.send_header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Tenant-ID")
        self.end_headers()

    def do_GET(self):
        if self.path == "/healthz":
            self._send(200, {"status": "ok"})
        else:
            self._send(200, {"service": os.getenv("OTEL_SERVICE_NAME", "stub")})

    def log_message(self, fmt, *args):
        pass


def run_stub(port: int | None = None) -> None:
    if port is None:
        port = int(os.getenv("PORT", "8000"))
    server = http.server.ThreadingHTTPServer(("0.0.0.0", port), _Handler)

    shutdown_flag = False

    def shutdown(signum, frame):
        nonlocal shutdown_flag
        if shutdown_flag:
            return
        shutdown_flag = True
        print("stub: shutting down...", flush=True)
        server.shutdown()

    signal.signal(signal.SIGINT, shutdown)
    signal.signal(signal.SIGTERM, shutdown)

    print(f"stub listening on :{port}", flush=True)
    server.serve_forever()
