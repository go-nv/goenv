# NOTE: Python3 is assumed
import http.server
import socketserver
import os
import sys

port = 8090

if len(sys.argv) == 2 and len(sys.argv[1]) > 0:
  port = int(sys.argv[1])

serve_dir = os.path.join(os.path.dirname(__file__), 'http-definitions')
os.chdir(serve_dir)

Handler = http.server.SimpleHTTPRequestHandler
httpd = socketserver.TCPServer(("localhost", port), Handler)
print("serving http-definitions from:", serve_dir, "and listening at port:", port)
httpd.serve_forever()
