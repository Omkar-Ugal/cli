import json
import os
from http.server import HTTPServer, BaseHTTPRequestHandler

# Defaults to the root disk; tests that need the counter to survive a stop or
# branch instead point this at a mounted volume, e.g. /data/counter.txt.
COUNTER_FILE = os.environ.get("COUNTER_FILE", "/counter.txt")

def load_counter():
    try:
        with open(COUNTER_FILE) as f:
            return int(f.read().strip())
    except (FileNotFoundError, ValueError):
        return 0

def save_counter(value):
    with open(COUNTER_FILE, "w") as f:
        f.write(str(value))

# Seeded from disk on boot so the counter survives a process restart as long
# as the underlying rootfs does (unlike an in-memory-only value, which is
# always lost on restart regardless of whether the disk was preserved).
counter = load_counter()
processed = {}

class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == "/count":
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            self.wfile.write(json.dumps({"count": counter}).encode())
        else:
            self.send_response(404)
            self.end_headers()

    def do_POST(self):
        global counter
        if self.path == "/increment":
            length = int(self.headers.get("Content-Length", 0))
            body = json.loads(self.rfile.read(length)) if length else {}
            key = self.headers.get("Idempotency-Key")
            if key and key in processed:
                response = processed[key]
            else:
                delta = body.get("delta", 1)
                if delta == "reset":
                    counter = 0
                else:
                    counter += int(delta)
                save_counter(counter)
                response = json.dumps({"count": counter}).encode()
                if key:
                    processed[key] = response
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            self.wfile.write(response)
        else:
            self.send_response(404)
            self.end_headers()

    def log_message(self, format, *args):
        pass  # silence request logging

HTTPServer(("", 8080), Handler).serve_forever()
