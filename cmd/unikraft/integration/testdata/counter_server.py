import json
from http.server import HTTPServer, BaseHTTPRequestHandler

counter = 0

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
            delta = body.get("delta", 1)
            if delta == "reset":
                counter = 0
            else:
                counter += int(delta)
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            self.wfile.write(json.dumps({"count": counter}).encode())
        else:
            self.send_response(404)
            self.end_headers()

    def log_message(self, format, *args):
        pass  # silence request logging

HTTPServer(("", 8080), Handler).serve_forever()
