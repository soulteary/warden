#!/usr/bin/env python3
"""One fixed owned draft recovery. No generic release adoption or image write."""
import argparse
import datetime
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import ssl
import stat
import subprocess
import sys
import tempfile
import urllib.error
import urllib.request

REPO = 'soulteary/warden'
TAG = 'v1.6.3'
SOURCE = '6fe54f24f78c28225882c2aae779e1020bb8f87e'
TREE = 'd98d9dad3f1b2ca916802301d8175da662246782'
TAG_OBJECT = 'e83fd755541be594032279d14ce4c610bb449da7'
INDEX = 'sha256:9ac7db4184d922ddaadfa99f9e65a59e88641f08181f6eab83a57c19a9e3234c'
DATE = '2026-11-09T12:51:58+0000'
RELEASE = 439146629
RUN = 42620110898
JOB = 129169750296
GO = '1.27.3'
SYFT = 'v1.51.1'
# Root binds these only after independent image qualification and source review.
# A caller-supplied boolean/file is never sufficient authorization.
ROOT_IMAGE_REVIEW_SHA256 = 'e4c212f42523abff96cd9671c38c9340c3eb946e1aff50ec7a1561eacfd6afd6'
ROOT_CAPSULE_SHA256 = '067a152ce3eec40aad17be038ccaf796be64a24271b7e7b6f7886270c8fe0457'
API = 'https://api.github.com/repos/' + REPO
REGISTRY = 'https://ghcr.io/v2/' + REPO
IDENTITY = 'https://github.com/' + REPO + '/.github/workflows/release.yml@refs/tags/' + TAG
ROOT = Path(__file__).resolve().parent
BINARIES = tuple('warden-' + system + '-' + arch + ('.exe' if system == 'windows' else '')
                 for system in ('linux', 'darwin', 'windows') for arch in ('amd64', 'arm64'))
FILES = BINARIES + ('warden-image-sbom.spdx.json', 'checksums.txt')
MAX_FILE = 128 * 1024 * 1024
MAX_TOTAL = 512 * 1024 * 1024
REASONS = frozenset({'authority-unbound', 'authority-hash', 'authority-shape', 'authority-evidence',
                    'http-status', 'http-shape', 'read-only-client', 'tag', 'source', 'original-attempt', 'draft',
                    'published', 'index', 'signature', 'filesystem', 'source-checkout',
                    'command-failed', 'command-unknown', 'build-record', 'asset', 'upload',
                    'publish', 'replay', 'unknown'})


class Refused(Exception):
    def __init__(self, reason, status=None, unknown=False):
        self.reason = reason if reason in REASONS else 'unknown'
        self.status = status if type(status) is int and 100 <= status <= 599 else None
        self.unknown = bool(unknown)
        super().__init__(self.reason)


STAGES = frozenset({'start', 'authority', 'tag-ref', 'tag-object', 'source', 'original-run',
                    'original-jobs', 'draft', 'draft-assets', 'published', 'registry-token',
                    'current-tag-index', 'immutable-index', 'cosign', 'source-checkout',
                    'go-version', 'go-mod-download', 'go-mod-verify', 'artifact-read', 'publish'} |
                   {'go-build-' + system + '-' + arch for system in ('linux','darwin','windows') for arch in ('amd64','arm64')} |
                   {'upload-' + name for name in FILES})
_STAGE = 'start'
_UPLOAD_STATE = {'expected_assets': {}, 'known_uploaded_assets': [], 'current_asset': None}


def mark(stage):
    global _STAGE
    _STAGE = stage if stage in STAGES else 'start'


def require(value, reason):
    if not value:
        raise Refused(reason)


def expect_status(status, expected, reason):
    if status != expected:
        raise Refused(reason, status=status)


def digest(raw):
    return hashlib.sha256(raw).hexdigest()


def canonical(value):
    return json.dumps(value, sort_keys=True, separators=(',', ':')).encode()


def decode(raw):
    def unique(pairs):
        value = {}
        for key, item in pairs:
            require(key not in value, 'http-shape')
            value[key] = item
        return value
    def nonfinite(value):
        raise Refused('http-shape')
    try:
        return json.loads(raw, object_pairs_hook=unique, parse_constant=nonfinite)
    except (ValueError, UnicodeError):
        raise Refused('http-shape') from None


def stamp(value):
    return (value.st_dev, value.st_ino, value.st_mode, value.st_uid, value.st_gid,
            value.st_nlink, value.st_size, value.st_mtime_ns, value.st_ctime_ns)


def read_file(path, limit=MAX_FILE):
    path = Path(path)
    require(path.is_absolute() and path.parent.resolve(strict=True) == path.parent, 'filesystem')
    parent = stamp(os.lstat(path.parent))
    fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK)
    try:
        before = os.fstat(fd)
        require(stat.S_ISREG(before.st_mode) and before.st_nlink == 1 and
                before.st_uid == os.geteuid() and not before.st_mode & 0o022 and
                0 < before.st_size <= limit, 'filesystem')
        raw = bytearray()
        while len(raw) < before.st_size:
            part = os.read(fd, min(65536, before.st_size - len(raw)))
            require(bool(part), 'filesystem')
            raw.extend(part)
        require(os.read(fd, 1) == b'' and stamp(before) == stamp(os.fstat(fd)) ==
                stamp(os.lstat(path)) and parent == stamp(os.lstat(path.parent)), 'filesystem')
        return bytes(raw)
    finally:
        os.close(fd)


def publish_json(directory, name, value):
    directory = Path(directory)
    require(directory.resolve(strict=True) == directory and directory.stat().st_uid == os.geteuid()
            and stat.S_IMODE(directory.stat().st_mode) == 0o700, 'filesystem')
    target = directory / name
    require(not target.exists() and not target.is_symlink(), 'replay')
    fd, temporary = tempfile.mkstemp(dir=directory, prefix='.pending-')
    try:
        with os.fdopen(fd, 'wb') as stream:
            stream.write(canonical(value) + b'\n')
            stream.flush()
            os.fsync(stream.fileno())
        # Atomic no-replace publication. Never truncate or adopt an old role.
        os.link(temporary, target, follow_symlinks=False)
    finally:
        os.unlink(temporary)


def attempt():
    return dict(repository=REPO, tag=TAG, source=SOURCE, tree=TREE, tag_object=TAG_OBJECT,
                original_run=RUN, original_job=JOB, draft_id=RELEASE, index_digest=INDEX,
                release_build_date=DATE, go_version=GO, syft_version=SYFT)


def authority():
    mark('authority')
    require(type(ROOT_IMAGE_REVIEW_SHA256) is str and type(ROOT_CAPSULE_SHA256) is str,
            'authority-unbound')
    review_raw = read_file(ROOT / 'image-qualification.json', 1024 * 1024)
    capsule_raw = read_file(ROOT / 'capsule.json', 1024 * 1024)
    require(digest(review_raw) == ROOT_IMAGE_REVIEW_SHA256 and
            digest(capsule_raw) == ROOT_CAPSULE_SHA256, 'authority-hash')
    review, capsule = decode(review_raw), decode(capsule_raw)
    require(isinstance(capsule, dict) and set(capsule) ==
            {'schema', 'attempt', 'image_qualification_sha256', 'recovery_nonce'}, 'authority-shape')
    require(capsule['schema'] == 'warden163-owned-recovery-capsule-v1' and
            capsule['attempt'] == attempt() and capsule['image_qualification_sha256'] ==
            ROOT_IMAGE_REVIEW_SHA256 and isinstance(capsule['recovery_nonce'], str) and
            re.fullmatch('[0-9a-f]{64}', capsule['recovery_nonce']), 'authority-shape')
    expected = dict(schema='warden163-partial-image-qualification-v1', index_digest=INDEX,
                    source_revision=SOURCE, tag_object=TAG_OBJECT, build_date=DATE,
                    platforms=['linux/amd64', 'linux/arm64'], user='10001:10001',
                    cosign_known_exit=0, source_constraints='exact-sha-ref-repository-push',
                    wrong_source_control='specific-workflow-sha-mismatch')
    require(isinstance(review, dict) and all(review.get(k) == v for k, v in expected.items()) and
            set(review) == set(expected) | {'raw_graph_manifest_sha256', 'cosign_capture_sha256'},
            'authority-evidence')
    for name in ('raw_graph_manifest_sha256', 'cosign_capture_sha256'):
        require(isinstance(review[name], str) and re.fullmatch('[0-9a-f]{64}', review[name]),
                'authority-evidence')
    return capsule


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *args, **kwargs):
        raise Refused('http-status')


class Client:
    def __init__(self, phase='plan'):
        require(phase in ('plan', 'inspect-partial', 'rebuild', 'upload'), 'replay')
        self.phase = phase
        self.token = os.environ.get('GH_TOKEN', '')
        require(bool(self.token) and not any(c in self.token for c in '\r\n'), 'http-shape')
        self.opener = urllib.request.build_opener(urllib.request.ProxyHandler({}), NoRedirect(),
                       urllib.request.HTTPSHandler(context=ssl.create_default_context()))

    def request(self, method, url, raw=None, registry_token=None, limit=8*1024*1024):
        # Draft visibility needs a write-capable token; non-upload phases still
        # cannot issue a mutating method or send a request body.
        require(self.phase == 'upload' or (method == 'GET' and raw is None),
                'read-only-client')
        require(url.startswith(API + '/') or url.startswith('https://uploads.github.com/repos/' + REPO + '/')
                or url.startswith(REGISTRY + '/') or url ==
                'https://ghcr.io/token?service=ghcr.io&scope=repository:' + REPO + ':pull', 'http-shape')
        require(method in ('GET', 'POST', 'PATCH'), 'http-shape')
        headers = {'Accept': 'application/json', 'User-Agent': 'warden163-fixed-recovery'}
        if url.startswith(API + '/') or url.startswith('https://uploads.github.com/'):
            headers['Authorization'] = 'Bearer ' + self.token
        if registry_token is not None:
            require(url.startswith(REGISTRY + '/') and isinstance(registry_token, str) and
                    0 < len(registry_token) <= 16384 and
                    all(33 <= ord(c) <= 126 for c in registry_token), 'http-shape')
            headers['Authorization'] = 'Bearer ' + registry_token
            headers['Accept'] = 'application/vnd.oci.image.index.v1+json, application/vnd.docker.distribution.manifest.list.v2+json'
        if raw is not None:
            require(type(raw) is bytes and len(raw) <= MAX_FILE, 'http-shape')
            headers['Content-Type'] = 'application/octet-stream' if '/assets?' in url else 'application/json'
        request = urllib.request.Request(url, data=raw, headers=headers, method=method)
        try:
            try:
                response = self.opener.open(request, timeout=30)
            except urllib.error.HTTPError as error:
                response = error
            with response:
                body = response.read(limit + 1)
                require(len(body) <= limit, 'http-shape')
                return response.code, body
        except Refused:
            raise
        except Exception:
            raise Refused('http-status', unknown=method != 'GET') from None


def get(client, path):
    stages = {'/git/ref/tags/' + TAG:'tag-ref', '/git/tags/' + TAG_OBJECT:'tag-object',
              '/commits/' + SOURCE:'source', '/actions/runs/' + str(RUN):'original-run',
              '/actions/runs/' + str(RUN) + '/jobs?per_page=100':'original-jobs',
              '/releases/' + str(RELEASE):'draft',
              '/releases/' + str(RELEASE) + '/assets?per_page=100':'draft-assets'}
    mark(stages.get(path, 'start'))
    status, raw = client.request('GET', API + path)
    if status != 200:
        raise Refused('http-status', status=status)
    return decode(raw)


def cosign_argv():
    return ['cosign', 'verify', 'ghcr.io/' + REPO + '@' + INDEX,
            '--certificate-identity', IDENTITY,
            '--certificate-oidc-issuer', 'https://token.actions.githubusercontent.com',
            '--certificate-github-workflow-sha', SOURCE,
            '--certificate-github-workflow-ref', 'refs/tags/' + TAG,
            '--certificate-github-workflow-repository', REPO,
            '--certificate-github-workflow-trigger', 'push']


def live_bindings(client, command, owned_assets=None):
    ref = get(client, '/git/ref/tags/' + TAG)
    require(ref.get('ref') == 'refs/tags/' + TAG and ref.get('object', {}).get('sha') == TAG_OBJECT
            and ref['object'].get('type') == 'tag', 'tag')
    tag = get(client, '/git/tags/' + TAG_OBJECT)
    require(tag.get('sha') == TAG_OBJECT and tag.get('tag') == TAG and
            tag.get('object', {}).get('sha') == SOURCE and tag['object'].get('type') == 'commit', 'tag')
    source = get(client, '/commits/' + SOURCE)
    require(source.get('sha') == SOURCE and source.get('commit', {}).get('tree', {}).get('sha') == TREE,
            'source')
    run = get(client, '/actions/runs/' + str(RUN))
    require(run.get('id') == RUN and run.get('head_sha') == SOURCE and run.get('event') == 'push'
            and run.get('head_branch') == TAG and run.get('run_attempt') == 1
            and run.get('status') == 'completed' and run.get('conclusion') == 'failure'
            and run.get('path') == '.github/workflows/release.yml', 'original-attempt')
    jobs = get(client, '/actions/runs/' + str(RUN) + '/jobs?per_page=100')
    require(type(jobs.get('total_count')) is int and 0 < jobs['total_count'] <= 100 and
            isinstance(jobs.get('jobs'), list) and len(jobs['jobs']) == jobs['total_count'], 'original-attempt')
    selected = [j for j in jobs.get('jobs', []) if j.get('id') == JOB]
    require(len(selected) == 1 and selected[0].get('status') == 'completed'
            and selected[0].get('conclusion') == 'failure', 'original-attempt')
    steps = selected[0].get('steps', [])
    require([s.get('number') for s in steps if s.get('conclusion') == 'failure'] == [23] and
            all(s.get('status') == 'completed' for s in steps if s.get('number') in (17,18,20,21,22,23)) and
            all(any(s.get('number') == n and s.get('conclusion') == 'success' for s in steps)
                for n in (17, 18, 20, 21, 22)), 'original-attempt')
    release = get(client, '/releases/' + str(RELEASE))
    require(release.get('id') == RELEASE and release.get('tag_name') == TAG and
            release.get('draft') is True and release.get('prerelease') is False and
            isinstance(release.get('assets'), list) and release.get('body') == 'See CHANGELOG.md at ' + SOURCE
            and release.get('author', {}).get('login') == 'github-actions[bot]', 'draft')
    listed = get(client, '/releases/' + str(RELEASE) + '/assets?per_page=100')
    require(isinstance(listed, list), 'draft')
    if owned_assets is None:
        require(release['assets'] == [] and listed == [], 'draft')
    else:
        # Only the in-memory results of this call's eight successful POSTs may
        # reach here. No persisted/tag-discovered partial set can be adopted.
        require(len(owned_assets) == 8 and {a['name'] for a in owned_assets} == set(FILES), 'asset')
        expected = {a['name']:a for a in owned_assets}
        for rows in (release['assets'], listed):
            require(len(rows) == 8 and {a.get('name') for a in rows} == set(FILES), 'asset')
            for item in rows:
                require({k:item.get(k) for k in ('id', 'name', 'size', 'digest')} == expected[item['name']], 'asset')
    mark('published')
    status, raw = client.request('GET', API + '/releases/tags/' + TAG)
    expect_status(status, 404, 'published')
    require(decode(raw).get('message') == 'Not Found', 'published')
    mark('registry-token')
    status, raw = client.request('GET', 'https://ghcr.io/token?service=ghcr.io&scope=repository:' + REPO + ':pull')
    token = decode(raw).get('token')
    expect_status(status, 200, 'index')
    require(isinstance(token, str) and bool(token), 'index')
    for reference in ('1.6.3', INDEX):
        mark('current-tag-index' if reference == '1.6.3' else 'immutable-index')
        status, raw = client.request('GET', REGISTRY + '/manifests/' + reference, registry_token=token)
        expect_status(status, 200, 'index')
        require('sha256:' + digest(raw) == INDEX, 'index')
    mark('cosign')
    require(command(cosign_argv(), 'cosign', None, None) == 0, 'signature')
    return dict(tag='pass', source='pass', original_attempt='pass', draft_empty='pass',
                published_absent='pass', current_index='pass', exact_source_signature='pass')


def minimal_env(extra=None):
    result = {name: os.environ[name] for name in ('PATH', 'HOME', 'GOMODCACHE', 'GOCACHE', 'TMPDIR')
              if name in os.environ}
    result.update(GOTOOLCHAIN='local', GOENV='off', GOWORK='off', CGO_ENABLED='0')
    if extra:
        result.update(extra)
    return result


class Commands:
    def __init__(self, directory, phase):
        require(phase in ('plan', 'inspect-partial', 'rebuild', 'upload'), 'replay')
        self.directory = Path(directory)
        self.phase = phase
        self.counter = 0

    def __call__(self, argv, stage, cwd, env):
        mark(stage)
        self.counter += 1
        logs = self.directory / ('logs-' + self.phase)
        logs.mkdir(mode=0o700, exist_ok=True)
        require(logs.resolve(strict=True) == logs and logs.stat().st_uid == os.geteuid()
                and stat.S_IMODE(logs.stat().st_mode) == 0o700, 'filesystem')
        path = logs / (str(self.counter) + '-' + stage + '.private')
        fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
        with os.fdopen(fd, 'wb') as output:
            require(stat.S_ISREG(os.fstat(output.fileno()).st_mode) and
                    os.fstat(output.fileno()).st_nlink == 1 and
                    stat.S_IMODE(os.fstat(output.fileno()).st_mode) == 0o600, 'filesystem')
            try:
                child_env = env or minimal_env()
                if stage == 'cosign':
                    config = self.directory / 'empty-docker-config'
                    config.mkdir(mode=0o700, exist_ok=True)
                    require(config.resolve(strict=True) == config and config.stat().st_uid == os.geteuid()
                            and stat.S_IMODE(config.stat().st_mode) == 0o700 and not list(config.iterdir()), 'filesystem')
                    child_env = dict(child_env, DOCKER_CONFIG=str(config))
                process = subprocess.Popen(argv, cwd=cwd, env=child_env,
                                           stdout=output, stderr=subprocess.STDOUT)
                code = process.wait(timeout=600)
            except subprocess.TimeoutExpired:
                # Local builder/reader status is unknown: retain PID/logs. No
                # kill, retry or automatic reuse is issued by this helper.
                publish_json(self.directory, 'unknown-command-' + str(self.counter) + '.json',
                             dict(stage=stage, pid=process.pid, outcome='unknown'))
                raise Refused('command-unknown', unknown=True) from None
        if code != 0:
            raise Refused('command-failed')
        if stage == 'go-version':
            require(read_file(path, 4096).strip() == b'go version go1.27.3 linux/amd64', 'build-record')
        return code


def source_checkout(path, command):
    mark('source-checkout')
    path = Path(path).resolve(strict=True)
    env = minimal_env({'GIT_NO_REPLACE_OBJECTS': '1', 'GIT_CONFIG_NOSYSTEM': '1',
                       'GIT_CONFIG_GLOBAL': '/dev/null'})
    def git(*args):
        result = subprocess.run(['git', '-c', 'core.fsmonitor=false', '-C', str(path)] + list(args),
                                env=env, capture_output=True, timeout=15, check=True).stdout
        require(len(result) <= 8*1024*1024, 'source-checkout')
        return result
    require(git('rev-parse', 'HEAD').decode().strip() == SOURCE and
            git('rev-parse', 'HEAD^{tree}').decode().strip() == TREE, 'source-checkout')
    require(git('status', '--porcelain') == b'', 'source-checkout')
    count = 0
    expected_files = set()
    for entry in git('ls-tree', '-rz', '--full-tree', SOURCE).split(b'\0'):
        if not entry:
            continue
        metadata, name = entry.split(b'\t', 1)
        mode, kind, identity = metadata.split()
        require(mode in (b'100644', b'100755') and kind == b'blob', 'source-checkout')
        relative = Path(name.decode('utf-8'))
        require(not relative.is_absolute() and all(x not in ('', '.', '..') for x in relative.parts), 'source-checkout')
        expected_files.add(relative.as_posix())
        raw = read_file(path / relative)
        require(bool(os.stat(path / relative).st_mode & 0o111) == (mode == b'100755'), 'source-checkout')
        actual = hashlib.sha1(b'blob ' + str(len(raw)).encode() + b'\0' + raw).hexdigest()
        require(actual == identity.decode(), 'source-checkout')
        count += 1
    actual_files = set()
    for item in path.rglob('*'):
        relative = item.relative_to(path)
        if relative.parts[0] == '.git':
            continue
        require(not item.is_symlink(), 'source-checkout')
        if item.is_file():
            actual_files.add(relative.as_posix())
    require(0 < count <= 2000 and actual_files == expected_files, 'source-checkout')
    return path


def execution_context():
    require(os.environ.get('GITHUB_REPOSITORY') == REPO and
            os.environ.get('GITHUB_EVENT_NAME') == 'workflow_dispatch' and
            os.environ.get('GITHUB_REF') == 'refs/heads/main' and
            os.environ.get('GITHUB_WORKFLOW_REF') == REPO + '/.github/workflows/recover-warden163-owned-attempt.yml@refs/heads/main' and
            re.fullmatch('[0-9a-f]{40}', os.environ.get('GITHUB_WORKFLOW_SHA', '')),
            'replay')


def state(directory):
    execution_context()
    value = decode(read_file(Path(directory) / 'intent.json', 65536))
    require(value.get('attempt') == attempt() and value.get('run_id') == os.environ.get('GITHUB_RUN_ID')
            and value.get('run_attempt') == '1' and os.environ.get('GITHUB_RUN_ATTEMPT') == '1', 'replay')
    return value


def plan(directory, client, command, require_authority=False):
    execution_context()
    bound, capsule, binding_reason = False, None, None
    try:
        capsule = authority()
        bound = True
    except Refused as error:
        binding_reason = error.reason
        if require_authority:
            raise
    checks = live_bindings(client, command)
    require(os.environ.get('GITHUB_RUN_ATTEMPT') == '1' and
            re.fullmatch('[0-9]+', os.environ.get('GITHUB_RUN_ID', '')), 'replay')
    value = dict(attempt=attempt(), run_id=os.environ['GITHUB_RUN_ID'], run_attempt='1',
                 authority_bound=bound, capsule_sha256=ROOT_CAPSULE_SHA256,
                 qualification_sha256=ROOT_IMAGE_REVIEW_SHA256,
                 recovery_nonce=capsule['recovery_nonce'] if capsule else None)
    publish_json(directory, 'intent.json', value)
    return dict(checks=checks, authority_bound=bound, authority_reason=binding_reason, remote_writes_performed=False,
                build_permitted=bound, missing=[] if bound else ['Root capsule/image qualification bindings'])


def rebuild(directory, source, client, command):
    authority()
    intent = state(directory)
    require(intent['authority_bound'] and intent['capsule_sha256'] == ROOT_CAPSULE_SHA256 and
            intent['qualification_sha256'] == ROOT_IMAGE_REVIEW_SHA256, 'authority-hash')
    live_bindings(client, command)
    source = source_checkout(source, command)
    publish_json(directory, 'build-start.json', dict(attempt=attempt(), state='started'))
    artifacts = Path(directory) / 'artifacts'
    artifacts.mkdir(mode=0o700)
    env = minimal_env({'GOMODCACHE': str(Path(directory)/'go-mod-cache'),
                       'GOCACHE': str(Path(directory)/'go-build-cache'),
                       'GOPROXY': 'https://proxy.golang.org', 'GOSUMDB': 'sum.golang.org'})
    command(['go', 'version'], 'go-version', source, env)
    command(['go', 'mod', 'download'], 'go-mod-download', source, env)
    command(['go', 'mod', 'verify'], 'go-mod-verify', source, env)
    flags = '-s -w -X github.com/soulteary/version-kit/v4.Version=1.6.3 -X github.com/soulteary/version-kit/v4.Commit=' + SOURCE + ' -X github.com/soulteary/version-kit/v4.BuildDate=' + DATE
    hashes = {}
    for name in BINARIES:
        os_name, arch = name[len('warden-'):].removesuffix('.exe').split('-')
        build_env = dict(env, GOOS=os_name, GOARCH=arch)
        command(['go', 'build', '-mod=readonly', '-ldflags', flags, '-o', str(artifacts/name), '.'],
                'go-build-' + os_name + '-' + arch, source, build_env)
        raw = read_file(artifacts/name)
        hashes[name] = dict(size=len(raw), sha256=digest(raw))
    source_checkout(source, command)
    record = dict(attempt=attempt(), recovery_tool_workflow_sha=os.environ['GITHUB_WORKFLOW_SHA'],
                  recovery_run_id=os.environ['GITHUB_RUN_ID'], compiler_version_checked=GO, actual_rebuild_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                  release_build_date_preserved=DATE, original_unpublished_bytes_equal=False,
                  binaries=hashes)
    publish_json(directory, 'rebuilt.json', record)
    return dict(binaries_rebuilt=len(hashes), original_bytes_equality_claimed=False, remote_writes_performed=False)


def assets(directory):
    mark('artifact-read')
    record = decode(read_file(Path(directory)/'rebuilt.json', 65536))
    require(record.get('attempt') == attempt() and record.get('original_unpublished_bytes_equal') is False
            and set(record.get('binaries', {})) == set(BINARIES), 'build-record')
    root = Path(directory)/'artifacts'
    require({p.name for p in root.iterdir()} == set(BINARIES) | {'warden-image-sbom.spdx.json'}, 'asset')
    payloads = {}
    for name in BINARIES + ('warden-image-sbom.spdx.json',):
        raw = read_file(root/name)
        if name in BINARIES:
            require(record['binaries'][name] == dict(size=len(raw), sha256=digest(raw)), 'build-record')
        else:
            sbom = decode(raw)
            require(isinstance(sbom, dict) and isinstance(sbom.get('spdxVersion'), str)
                    and sbom['spdxVersion'].startswith('SPDX-'), 'asset')
        payloads[name] = raw
    checksums = ''.join(digest(payloads[name]) + '  ' + name + '\n' for name in sorted(payloads)).encode()
    fd = os.open(root/'checksums.txt', os.O_CREAT | os.O_EXCL | os.O_WRONLY | os.O_NOFOLLOW, 0o600)
    with os.fdopen(fd, 'wb') as output:
        output.write(checksums)
        output.flush()
        os.fsync(output.fileno())
    payloads['checksums.txt'] = read_file(root/'checksums.txt')
    require(sum(map(len, payloads.values())) <= MAX_TOTAL, 'asset')
    return payloads


def upload(directory, client, command):
    authority()
    intent = state(directory)
    require(intent.get('authority_bound') and intent.get('capsule_sha256') == ROOT_CAPSULE_SHA256
            and intent.get('qualification_sha256') == ROOT_IMAGE_REVIEW_SHA256, 'authority-hash')
    payloads = assets(directory)
    _UPLOAD_STATE['expected_assets'] = {n:dict(size=len(r), sha256=digest(r)) for n,r in payloads.items()}
    live_bindings(client, command)
    publish_json(directory, 'upload-start.json', dict(attempt=attempt(), state='started',
                 assets={name:dict(size=len(raw), sha256=digest(raw)) for name, raw in payloads.items()}))
    observed = []
    for name in FILES:
        mark('upload-' + name)
        _UPLOAD_STATE['current_asset'] = name
        status = None
        try:
            status, raw = client.request('POST', 'https://uploads.github.com/repos/' + REPO + '/releases/' + str(RELEASE) + '/assets?name=' + name, payloads[name])
            expect_status(status, 201, 'upload')
            asset = decode(raw)
            require(type(asset.get('id')) is int and asset['id'] > 0 and asset.get('name') == name
                    and asset.get('size') == len(payloads[name]) and
                    asset.get('digest') == 'sha256:' + digest(payloads[name]) and asset.get('state') == 'uploaded', 'asset')
            observed.append({key:asset[key] for key in ('id', 'name', 'size', 'digest')})
            _UPLOAD_STATE['known_uploaded_assets'] = observed.copy()
            publish_json(directory, 'uploaded-' + name + '.json', observed[-1])
        except Exception:
            # The request may have completed remotely even if its response or
            # local result publication failed. Never proceed, retry or clean up.
            raise Refused('upload', status=status if 'status' in locals() else None, unknown=True) from None
    _UPLOAD_STATE['current_asset'] = None
    release = get(client, '/releases/' + str(RELEASE))
    listed = get(client, '/releases/' + str(RELEASE) + '/assets?per_page=100')
    require(release.get('id') == RELEASE and release.get('draft') is True and release.get('tag_name') == TAG
            and isinstance(listed, list) and len(listed) == 8, 'draft')
    expected = {a['name']:a for a in observed}
    require(len(expected) == 8 and {a.get('name') for a in listed} == set(FILES), 'asset')
    for a in listed:
        require({key:a.get(key) for key in ('id', 'name', 'size', 'digest')} == expected[a['name']], 'asset')
    # Recheck all immutable bindings after exact own-asset readback.
    authority()
    live_bindings(client, command, owned_assets=observed)
    publish_json(directory, 'publish-intent.json', dict(release_id=RELEASE, assets=observed))
    mark('publish')
    status = None
    try:
        status, raw = client.request('PATCH', API + '/releases/' + str(RELEASE), canonical({'draft':False}))
        expect_status(status, 200, 'publish')
        final = decode(raw)
        require(final.get('id') == RELEASE and final.get('tag_name') == TAG
                and final.get('draft') is False, 'publish')
    except Exception:
        raise Refused('publish', status=status if 'status' in locals() else None, unknown=True) from None
    publish_json(directory, 'published.json', dict(release_id=RELEASE, assets=observed,
                 image_republished=False, original_bytes_equal_claimed=False))
    return dict(release_id=RELEASE, assets_uploaded=8, published=True, image_republished=False,
                original_bytes_equality_claimed=False)


def inspect_partial(client):
    # Read-only reconciliation facts, never a route around empty-draft admission.
    release = get(client, '/releases/' + str(RELEASE))
    listed = get(client, '/releases/' + str(RELEASE) + '/assets?per_page=100')
    require(release.get('id') == RELEASE and release.get('tag_name') == TAG and
            type(release.get('draft')) is bool and isinstance(listed, list) and len(listed) <= 8, 'draft')
    observed = []
    for item in listed:
        require(type(item.get('id')) is int and item['id'] > 0 and item.get('name') in FILES and
                type(item.get('size')) is int and 0 < item['size'] <= MAX_FILE and
                isinstance(item.get('digest'), str) and re.fullmatch('sha256:[0-9a-f]{64}', item['digest']), 'asset')
        observed.append({k:item[k] for k in ('id', 'name', 'size', 'digest')})
    require(len({a['name'] for a in observed}) == len(observed), 'asset')
    return dict(release_id=RELEASE, draft=release['draft'], current_assets=observed,
                ownership_or_execution_authority_inferred=False, retry_authorized=False,
                remote_writes_performed=False, separate_partial_state_review_required=bool(observed))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--phase', choices=('plan', 'rebuild', 'upload', 'inspect-partial'), required=True)
    parser.add_argument('--output', required=True)
    parser.add_argument('--source')
    parser.add_argument('--require-authorization', action='store_true')
    args = parser.parse_args()
    directory = Path(args.output)
    report = dict(schema='warden163-owned-recovery-safe-report-v1', phase=args.phase, success=False,
                  reason=None, actual_attempt=attempt(), partial_state_must_be_retained=True)
    try:
        require(directory.is_absolute() and directory.parent.resolve(strict=True) == directory.parent, 'filesystem')
        if args.phase in ('plan', 'inspect-partial'):
            directory.mkdir(mode=0o700)
        else:
            require(directory.is_dir() and stat.S_IMODE(directory.stat().st_mode) == 0o700
                    and directory.stat().st_uid == os.geteuid(), 'filesystem')
        client, command = Client(args.phase), Commands(directory, args.phase)
        if args.phase == 'plan':
            report['result'] = plan(directory, client, command, args.require_authorization)
        elif args.phase == 'inspect-partial':
            report['result'] = inspect_partial(client)
        elif args.phase == 'rebuild':
            require(args.source is not None, 'source-checkout')
            report['result'] = rebuild(directory, args.source, client, command)
        else:
            report['result'] = upload(directory, client, command)
        report['success'] = True
        code = 0
    except Exception as error:
        report['stage'] = _STAGE
        report['reason'] = error.reason if isinstance(error, Refused) else 'unknown'
        report['http_status'] = error.status if isinstance(error, Refused) else None
        report['unknown_remote_or_command_state'] = error.unknown if isinstance(error, Refused) else True
        report['upload_facts'] = _UPLOAD_STATE
        code = 1
    if args.phase == 'upload':
        report['upload_facts'] = _UPLOAD_STATE
    try:
        publish_json(directory, 'safe-report-' + args.phase + '.json', report)
    except Exception:
        print('recovery report publication refused', file=sys.stderr)
        return 1
    print(json.dumps(report, sort_keys=True))
    return code


if __name__ == '__main__':
    raise SystemExit(main())
