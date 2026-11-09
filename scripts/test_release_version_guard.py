#!/usr/bin/env python3
"""Offline protocol and actual workflow-shell regressions; no network/builds."""
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

import yaml

ROOT = Path(__file__).resolve().parents[1]
spec = importlib.util.spec_from_file_location('guard', ROOT / 'scripts/release_version_guard.py')
guard = importlib.util.module_from_spec(spec)
spec.loader.exec_module(guard)
created_spec = importlib.util.spec_from_file_location('created', ROOT / 'scripts/release_created_id.py')
created = importlib.util.module_from_spec(created_spec)
created_spec.loader.exec_module(created)
SOURCE = 'a' * 40


class GuardTest(unittest.TestCase):
    def replies(self, replacements=None):
        values = [(404, {'message': 'Not Found'}), (200, []), (200, {'token': 'REGISTRY_CANARY'}),
                  (404, {'errors': [{'code': 'MANIFEST_UNKNOWN'}]})]
        for key, value in (replacements or {}).items():
            values[key] = value
        calls = []
        def fetch(url, headers):
            calls.append((url, headers))
            status, body = values[len(calls) - 1]
            return status, body if isinstance(body, bytes) else json.dumps(body).encode()
        return fetch, calls

    def test_exact_absence_and_authorization_origins(self):
        fetch, calls = self.replies()
        result = guard.preflight('v1.6.3', SOURCE, 'GITHUB_CANARY', fetch)
        self.assertTrue(result['version_image_absent'])
        self.assertEqual(calls[2][1], {})
        self.assertEqual(calls[3][1]['Authorization'], 'Bearer REGISTRY_CANARY')
        self.assertNotIn('GITHUB_CANARY', repr(calls[2:]))
        self.assertTrue(calls[3][0].endswith('/manifests/1.6.3'))

    def test_existing_or_unknown_never_absent(self):
        for stage in (0, 3):
            for status in (200, 201, 301, 401, 403, 429, 500):
                with self.subTest(stage=stage, status=status):
                    fetch, _ = self.replies({stage: (status, {})})
                    with self.assertRaises(guard.Refused):
                        guard.preflight('v1.6.3', SOURCE, 'secret', fetch)
        for stage, body in ((0, b'<html>404</html>'), (3, {'errors': []}),
                            (3, {'errors': [{'code': 'UNAUTHORIZED'}]}), (2, {'token': ''})):
            fetch, _ = self.replies({stage: (404 if stage != 2 else 200, body)})
            with self.assertRaises(guard.Refused):
                guard.preflight('v1.6.3', SOURCE, 'secret', fetch)

    def test_existing_draft_or_incomplete_inventory_refuses(self):
        for status, body in ((200, [{'tag_name': 'v1.6.3', 'draft': True}]),
                             (403, []), (200, [{}]), (200, b'not-json')):
            fetch, calls = self.replies({1: (status, body)})
            with self.assertRaises(guard.Refused):
                guard.preflight('v1.6.3', SOURCE, 'secret', fetch)
            self.assertEqual(len(calls), 2)
        calls = []
        def many(url, headers):
            calls.append(url)
            return (404, b'{"message":"Not Found"}') if len(calls) == 1 else (200, json.dumps([{'tag_name': 'v0.0.1'}] * 20).encode())
        with self.assertRaises(guard.Refused):
            guard.preflight('v1.6.3', SOURCE, 'secret', many)
        self.assertEqual(len(calls), 51)

    def test_invalid_inputs_never_request(self):
        for tag, source, token in (('../x', SOURCE, 's'), ('v1.6.3', 'a'*7, 's'),
                                   ('v1.6.3', SOURCE, ''), ('v1.6.3', SOURCE, 'bad\ntoken')):
            fetch, calls = self.replies()
            with self.assertRaises(guard.Refused):
                guard.preflight(tag, source, token, fetch)
            self.assertEqual(calls, [])

    def test_errors_are_fixed_and_redirects_forbidden(self):
        with self.assertRaises(guard.Refused):
            guard.NoRedirect().redirect_request(None, None, None, None, None, None)
        import io
        stderr = io.StringIO()
        with patch.object(guard, 'preflight', side_effect=RuntimeError('CANARY_URL_TOKEN')), \
             patch('sys.argv', ['guard', '--tag', 'v1.6.3', '--source-revision', SOURCE]), \
             patch('sys.stderr', stderr):
            self.assertEqual(guard.main(), 1)
        self.assertEqual(stderr.getvalue(), 'release version preflight refused\n')

    def test_response_budget_and_no_proxy(self):
        class Response:
            code = 200
            def __enter__(self): return self
            def __exit__(self, *a): pass
            def read(self, count):
                self.count = count
                return b'x' * count
        response = Response()
        class Opener:
            def open(self, req, timeout):
                self.timeout = timeout
                return response
        captured = []
        def builder(*handlers):
            captured.extend(handlers)
            return Opener()
        with patch.object(guard.urllib.request, 'build_opener', side_effect=builder):
            with self.assertRaises(guard.Refused):
                guard.request('https://api.github.com/', {})
        self.assertEqual(response.count, guard.LIMIT + 1)
        self.assertEqual(captured[0].proxies, {})
        self.assertIsInstance(captured[1], guard.NoRedirect)


class CreatedResponseTest(unittest.TestCase):
    def test_strict_status_header_body_and_budgets(self):
        import io
        body = json.dumps(dict(id=123, tag_name='v1.6.3', draft=True)).encode()
        for separator in (b'\n', b'\r\n'):
            response = separator.join([b'HTTP/2.0 201 Created', b'Content-Type: application/json', b'', body])
            self.assertEqual(created.created_id(io.BytesIO(response), 'v1.6.3'), 123)
        mixed = b'HTTP/2.0 201 Created\nContent-Type: application/json\r\n\r\n' + body
        self.assertEqual(created.created_id(io.BytesIO(mixed), 'v1.6.3'), 123)
        valid = b'HTTP/2.0 201 Created\nContent-Type: application/json\n\n' + body
        for response in (body, valid.replace(b'201', b'200', 1), valid.replace(b'201', b'202', 1),
                         valid.replace(b'application/json', b'text/html'), valid + b'TAIL',
                         valid.replace(b'123', b'true'), valid.replace(b'v1.6.3', b'v0.0.1'),
                         valid.replace(b'Content-Type:', b'Bad Header:'), valid.replace(b'123', b'NaN'),
                         valid.replace(b'{', b'{\"id\":999,', 1), b'x'*(created.LIMIT+1),
                         b'HTTP/2.0 201 Created\nX: '+b'x'*32768+b'\nContent-Type: application/json\n\n'+body):
            with self.subTest(response_prefix=response[:30]):
                with self.assertRaises(Exception):
                    created.created_id(io.BytesIO(response), 'v1.6.3')


class WorkflowTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.raw = (ROOT / '.github/workflows/release.yml').read_text()
        cls.workflow = yaml.safe_load(cls.raw)
        cls.steps = cls.workflow['jobs']['release']['steps']

    def test_publication_source_and_order(self):
        self.assertFalse(self.workflow['concurrency']['cancel-in-progress'])
        self.assertIn('inputs.release_tag || github.ref_name', self.workflow['concurrency']['group'])
        self.assertNotIn('gh release delete', self.raw)
        self.assertNotIn('action-gh-release@', self.raw)
        self.assertNotIn('--clobber', self.raw)
        self.assertNotIn('rev-parse --short', self.raw)
        guards = [n for n, s in enumerate(self.steps) if 'release_version_guard.py' in s.get('run', '')]
        push = next(n for n, s in enumerate(self.steps) if s.get('id') == 'build-push')
        self.assertEqual(len(guards), 2)
        self.assertLess(guards[-1], push)
        self.assertEqual(self.steps[0]['with']['ref'], '${{ github.workflow_sha }}')
        self.assertFalse(self.steps[0]['with']['persist-credentials'])
        self.assertEqual(self.workflow['jobs']['release']['defaults']['run']['working-directory'], 'release-source')
        build = next(s for s in self.steps if s.get('id') == 'build-push')['with']
        self.assertEqual(build['context'], './release-source')
        self.assertIn('${{ steps.version.outputs.version }}', build['tags'].splitlines()[0])
        self.assertIn('source_revision', build['labels'])
        self.assertNotIn('steps.meta.outputs.labels', build['labels'])
        binaries = next(s['run'] for s in self.steps if 'LDFLAGS=' in s.get('run', ''))
        self.assertIn('steps.version.outputs.build_date', binaries)
        self.assertNotIn('date -u', binaries)

    def test_all_real_shell_blocks_parse(self):
        for step in self.steps:
            if 'run' in step:
                subprocess.run(['bash', '-n'], input=step['run'], text=True, check=True, capture_output=True)

    def run_publish(self, collision=False, upload_failure=False, http_status=201):
        body = next(s['run'] for s in self.steps if s['name'].startswith('Create release only'))
        with tempfile.TemporaryDirectory() as temp:
            workspace = Path(temp)
            root = workspace/'release-source'
            root.mkdir()
            tools = workspace/'release-tools/scripts'
            tools.mkdir(parents=True)
            (tools/'release_created_id.py').write_bytes((ROOT/'scripts/release_created_id.py').read_bytes())
            (root/'dist').mkdir()
            for name in ('warden-linux-amd64', 'warden-linux-arm64', 'warden-darwin-amd64', 'warden-darwin-arm64', 'warden-windows-amd64.exe', 'warden-windows-arm64.exe', 'warden-image-sbom.spdx.json', 'checksums.txt'):
                (root/'dist'/name).write_text('synthetic')
            (root/'gh').write_text('''#!/usr/bin/env python3
import json,os,sys
with open('calls','a') as f:f.write(json.dumps(sys.argv[1:])+'\\n')
a=sys.argv
method=a[a.index('--method')+1]
endpoint=a[a.index('--method')+2]
if method=='POST' and endpoint.endswith('/releases'):
 if os.environ['COLLISION']=='1':sys.exit(1)
 print('HTTP/2.0 '+os.environ['HTTP_STATUS']+(' Created' if os.environ['HTTP_STATUS']=='201' else ' Other')+chr(10)+'Content-Type: application/json'+chr(10)*2+json.dumps(dict(id=123,tag_name=os.environ['RELEASE_TAG'],draft=True)))
elif 'uploads.github.com' in ' '.join(a) and os.environ['UPLOAD_FAILURE']=='1':sys.exit(1)
else:print('{}')
''')
            (root/'gh').chmod(0o700)
            env = dict(os.environ, PATH=str(root)+os.pathsep+os.environ['PATH'], GITHUB_REPOSITORY='soulteary/warden', RELEASE_TAG='v1.6.3', SOURCE_REVISION=SOURCE, COLLISION=str(int(collision)), UPLOAD_FAILURE=str(int(upload_failure)), HTTP_STATUS=str(http_status))
            result = subprocess.run(['bash', '-c', body], cwd=root, env=env, capture_output=True, text=True)
            calls = [json.loads(x) for x in (root/'calls').read_text().splitlines()]
            return result, calls

    def test_create_collision_stops_without_update_or_upload(self):
        result, calls = self.run_publish(collision=True)
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(len(calls), 1)
        self.assertEqual(calls[0][:4], ['api','--include','--method','POST'])

    def test_zero_exit_non201_never_authorizes_existing_id(self):
        for status in (200, 202, 301, 302, 307, 308):
            with self.subTest(status=status):
                result, calls = self.run_publish(http_status=status)
                self.assertNotEqual(result.returncode, 0)
                self.assertEqual(len(calls), 1)
                self.assertEqual(result.stderr, 'release create response refused: http-status\n')

    def test_new_release_uploads_then_only_own_id_is_finalized(self):
        result, calls = self.run_publish()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(len(calls), 10)
        self.assertTrue(all('/releases/123/assets?' in ' '.join(c) for c in calls[1:9]))
        self.assertEqual(calls[-1][3], 'repos/soulteary/warden/releases/123')
        self.assertEqual(calls[-1][2], 'PATCH')
        result, calls = self.run_publish(upload_failure=True)
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(any(c[2] == 'PATCH' for c in calls))


if __name__ == '__main__':
    unittest.main()
