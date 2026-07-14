"""Shared M0 stub HTTP server for Python services.

Each service replaces this with its real implementation per ROADMAP.md
milestones. The service Dockerfiles vendor this file as ``_stub.py`` next
to their ``server.py``, so a service entrypoint is just::

    import _stub
    _stub.run_stub()
"""

from __future__ import annotations

import http.server
import json
import logging
import os
import signal
import sys
import threading


logger = logging.getLogger("stub")

_CORS_ORIGIN = os.getenv("CORS_ORIGIN", "http://localhost:3000")
_OTEL_SERVICE_NAME = os.getenv("OTEL_SERVICE_NAME", "stub")

_PORT_ENV = os.getenv("PORT", "8000")
try:
    _DEFAULT_PORT = int(_PORT_ENV)
    if not (0 < _DEFAULT_PORT <= 65535):
        raise ValueError
except ValueError:
    _DEFAULT_PORT = 8000
    logger.warning("invalid PORT=%r, falling back to %d", _PORT_ENV, _DEFAULT_PORT)


class _Handler(http.server.BaseHTTPRequestHandler):
    def _send(self, code, payload):
        body = json.dumps(payload).encode()
        try:
            self.send_response(code)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(body)))
            self.send_header("Access-Control-Allow-Origin", _CORS_ORIGIN)
            self.send_header("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
            self.send_header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Tenant-ID")
            self.end_headers()
            self.wfile.write(body)
        except (BrokenPipeError, ConnectionResetError, ConnectionAbortedError):
            pass

    def do_OPTIONS(self):
        try:
            self.send_response(204)
            self.send_header("Access-Control-Allow-Origin", _CORS_ORIGIN)
            self.send_header("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
            self.send_header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Tenant-ID")
            self.end_headers()
        except (BrokenPipeError, ConnectionResetError, ConnectionAbortedError):
            pass

    def do_GET(self):
        if self.path == "/healthz":
            self._send(200, {"status": "ok"})
        elif self.path == "/":
            self._send(200, {"service": _OTEL_SERVICE_NAME})
        else:
            self._send(404, {"error": "not found"})

    def log_message(self, fmt, *args):
        pass


def run_stub(port: int | None = None) -> None:
    if port is None:
        port = _DEFAULT_PORT

    server = http.server.ThreadingHTTPServer(("0.0.0.0", port), _Handler)

    shutdown_lock = threading.Lock()
    shutdown_flag = False

    def shutdown(signum, frame):
        nonlocal shutdown_flag
        with shutdown_lock:
            if shutdown_flag:
                return
            shutdown_flag = True
        logger.info("stub: shutting down...")
        server.shutdown()

    signal.signal(signal.SIGINT, shutdown)
    signal.signal(signal.SIGTERM, shutdown)

    logger.info("stub listening on :%d", port)
    server.serve_forever()


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, stream=sys.stdout)
    run_stub()
