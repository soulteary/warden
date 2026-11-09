#!/usr/bin/env python3
"""Fail-closed public-version preflight; never creates or changes a release."""
import argparse
import json
import os
import re
import sys
import urllib.error
import urllib.request

REPOSITORY = "soulteary/warden"
LIMIT = 1024 * 1024


class Refused(Exception):
    pass


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *args, **kwargs):
        raise Refused("redirect")


def request(url, headers):
    # Ignore ambient proxy settings and never forward credentials via redirects.
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}), NoRedirect())
    req = urllib.request.Request(url, headers=headers)
    try:
        response = opener.open(req, timeout=15)
    except urllib.error.HTTPError as error:
        response = error
    with response:
        raw = response.read(LIMIT + 1)
        if len(raw) > LIMIT:
            raise Refused("response size")
        return response.code, raw


def decode(raw):
    try:
        value = json.loads(raw)
    except (ValueError, UnicodeError):
        raise Refused("response JSON") from None
    if not isinstance(value, dict):
        raise Refused("response shape")
    return value


def preflight(tag, source, token, fetch=request):
    if not re.fullmatch(r"v[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?", tag):
        raise Refused("tag")
    if not re.fullmatch(r"[0-9a-f]{40}", source) or not token or any(c in token for c in "\r\n"):
        raise Refused("source or API authorization")
    status, raw = fetch(f"https://api.github.com/repos/{REPOSITORY}/releases/tags/{tag}",
                        {"Authorization": "Bearer " + token, "Accept": "application/vnd.github+json"})
    value = decode(raw)
    if status != 404 or value.get("message") != "Not Found":
        raise Refused("release exists or absence unproved")
    # The tag endpoint covers published releases only. Authenticated listing
    # also covers drafts visible to this contents:write workflow token.
    for page in range(1, 51):
        status, raw = fetch(f"https://api.github.com/repos/{REPOSITORY}/releases?per_page=20&page={page}",
                            {"Authorization": "Bearer " + token, "Accept": "application/vnd.github+json"})
        try:
            releases = json.loads(raw)
        except (ValueError, UnicodeError):
            raise Refused("release inventory JSON") from None
        if status != 200 or not isinstance(releases, list) or len(releases) > 20:
            raise Refused("release inventory")
        for release in releases:
            if not isinstance(release, dict) or not isinstance(release.get("tag_name"), str) or not release["tag_name"]:
                raise Refused("release inventory shape")
            if release["tag_name"] == tag:
                raise Refused("release exists")
        if len(releases) < 20:
            break
    else:
        raise Refused("release inventory budget")
    status, raw = fetch(f"https://ghcr.io/token?service=ghcr.io&scope=repository:{REPOSITORY}:pull", {})
    value = decode(raw)
    bearer = value.get("token")
    if status != 200 or not isinstance(bearer, str) or not bearer or any(c in bearer for c in "\r\n"):
        raise Refused("registry authorization")
    status, raw = fetch(f"https://ghcr.io/v2/{REPOSITORY}/manifests/{tag[1:]}",
                        {"Authorization": "Bearer " + bearer,
                         "Accept": "application/vnd.oci.image.index.v1+json, application/vnd.docker.distribution.manifest.list.v2+json, application/vnd.oci.image.manifest.v1+json, application/vnd.docker.distribution.manifest.v2+json"})
    value = decode(raw)
    errors = value.get("errors")
    if status != 404 or not isinstance(errors, list) or len(errors) != 1 or not isinstance(errors[0], dict) or errors[0].get("code") != "MANIFEST_UNKNOWN":
        raise Refused("image exists or absence unproved")
    return {"tag": tag, "source_revision": source, "release_absent": True, "version_image_absent": True}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--tag", required=True)
    parser.add_argument("--source-revision", required=True)
    args = parser.parse_args()
    try:
        result = preflight(args.tag, args.source_revision, os.environ.get("GH_TOKEN", ""))
    except Exception:
        # Never emit exception URLs, response bodies, tokens or raw server errors.
        print("release version preflight refused", file=sys.stderr)
        return 1
    print(json.dumps(result, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
