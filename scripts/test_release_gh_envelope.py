#!/usr/bin/env python3
"""Real installed gh against an owned loopback-only synthetic HTTP response."""
import importlib.util
import io
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import threading
import unittest
from http.server import BaseHTTPRequestHandler, HTTPServer

spec = importlib.util.spec_from_file_location('created', Path(__file__).with_name('release_created_id.py'))
created = importlib.util.module_from_spec(spec)
spec.loader.exec_module(created)


class GhEnvelopeTest(unittest.TestCase):
    def capture(self, status=201):
        gh = shutil.which('gh')
        self.assertIsNotNone(gh, 'Installed gh is required; do not skip the actual renderer regression')
        calls = []
        body = json.dumps({'id': 123, 'tag_name': 'v0.0.0-test', 'draft': True}).encode()

        class Handler(BaseHTTPRequestHandler):
            protocol_version = 'HTTP/1.1'

            def log_message(self, *args):
                pass

            def do_POST(self):
                length = int(self.headers.get('Content-Length', '0'))
                if not 0 < length <= 4096:
                    self.send_error(400)
                    return
                request = self.rfile.read(length)
                calls.append((self.command, self.path, json.loads(request)))
                self.send_response(status)
                self.send_header('Content-Type', 'application/json; charset=utf-8')
                self.send_header('Content-Length', str(len(body)))
                self.end_headers()
                self.wfile.write(body)

        with tempfile.TemporaryDirectory() as temp, HTTPServer(('127.0.0.1', 0), Handler) as server:
            config = Path(temp)/'gh-config'
            config.mkdir(mode=0o700)
            request = Path(temp)/'request.json'
            request.write_text('{"tag_name":"v0.0.0-test","draft":true}')
            thread = threading.Thread(target=server.serve_forever, daemon=True)
            thread.start()
            try:
                env = {'PATH': os.pathsep.join([str(Path(gh).parent), '/usr/bin', '/bin']),
                       'GH_CONFIG_DIR': str(config), 'GH_TOKEN': 'SYNTHETIC_LOOPBACK_ONLY_TOKEN',
                       'GH_PROMPT_DISABLED': '1', 'GH_NO_UPDATE_NOTIFIER': '1',
                       'GH_NO_EXTENSION_UPDATE_NOTIFIER': '1'}
                result = subprocess.run([gh, 'api', '--include', '--method', 'POST',
                                         f'http://127.0.0.1:{server.server_port}/repos/synthetic/release-test/releases',
                                         '--input', str(request)], cwd=temp, env=env,
                                        capture_output=True, timeout=10)
            finally:
                server.shutdown()
                thread.join(timeout=5)
                self.assertFalse(thread.is_alive(), 'Owned loopback fixture did not stop')
        self.assertEqual(calls, [('POST', '/repos/synthetic/release-test/releases',
                                  {'tag_name': 'v0.0.0-test', 'draft': True})])
        self.assertEqual(result.returncode, 0, result.stderr.decode(errors='replace'))
        self.assertEqual(result.stderr, b'')
        return result.stdout

    def test_actual_gh_mixed_envelope_preserves_explicit_201(self):
        raw = self.capture()
        # Official gh processResponse uses LF status + CRLF printHeaders. The
        # actual installed executable must demonstrate this seam, not a model.
        self.assertTrue(raw.startswith(b'HTTP/1.1 201 Created\n'))
        self.assertIn(b'Content-Type: application/json; charset=utf-8\r\n', raw)
        self.assertIn(b'\r\n\r\n', raw)
        self.assertEqual(created.created_id(io.BytesIO(raw), 'v0.0.0-test'), 123)
        self.assertNotIn(b'SYNTHETIC_LOOPBACK_ONLY_TOKEN', raw)

    def test_actual_gh_success_exit_200_is_still_refused(self):
        raw = self.capture(200)
        with self.assertRaises(created.Refused) as error:
            created.created_id(io.BytesIO(raw), 'v0.0.0-test')
        self.assertEqual(error.exception.reason, 'http-status')


if __name__ == '__main__':
    unittest.main()
