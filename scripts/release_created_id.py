#!/usr/bin/env python3
"""Authorize only an explicit HTTP 201 create response, never an existing ID."""
import argparse
import json
import re
import sys

LIMIT = 1024 * 1024
REASONS = frozenset({'response-budget', 'header-framing', 'header-budget', 'http-status',
                     'header-syntax', 'media-type', 'json-syntax', 'json-duplicate',
                     'json-nonfinite', 'record-shape', 'record-binding', 'unknown'})


class Refused(ValueError):
    def __init__(self, reason):
        self.reason = reason if reason in REASONS else 'unknown'
        super().__init__(self.reason)


def created_id(stream, tag):
    raw = stream.read(LIMIT + 1)
    if len(raw) > LIMIT:
        raise Refused('response-budget')
    # gh prints the status line with Fprintln (LF), headers with CRLF, then
    # a CRLF separator. Parse lines without modifying any response body bytes.
    lines = []
    cursor = 0
    while True:
        end = raw.find(b'\n', cursor)
        if end < 0:
            raise Refused('header-framing')
        line = raw[cursor:end]
        if line.endswith(b'\r'):
            line = line[:-1]
        if b'\r' in line:
            raise Refused('header-framing')
        cursor = end + 1
        if cursor > 32768 or len(lines) > 128:
            raise Refused('header-budget')
        if not line:
            break
        lines.append(line)
    if not lines or not re.fullmatch(rb'HTTP/(?:1\.[01]|2(?:\.0)?) 201(?: Created)?', lines[0]):
        raise Refused('http-status')
    content_types = []
    for line in lines[1:]:
        match = re.fullmatch(rb'([!#$%&\x27*+.^_`|~0-9A-Za-z-]+):[ \t]*([\x20-\x7e\t]*)', line)
        if not match:
            raise Refused('header-syntax')
        if match[1].lower() == b'content-type':
            content_types.append(match[2].lower().strip())
    if len(content_types) != 1 or content_types[0].split(b';', 1)[0].strip() != b'application/json':
        raise Refused('media-type')

    def unique_object(pairs):
        result = {}
        for key, item in pairs:
            if key in result:
                raise Refused('json-duplicate')
            result[key] = item
        return result

    def reject_constant(value):
        raise Refused('json-nonfinite')

    try:
        value = json.loads(raw[cursor:], object_pairs_hook=unique_object, parse_constant=reject_constant)
    except (json.JSONDecodeError, UnicodeError):
        raise Refused('json-syntax') from None
    if not isinstance(value, dict) or type(value.get('id')) is not int or value['id'] <= 0:
        raise Refused('record-shape')
    if value.get('tag_name') != tag or value.get('draft') is not True:
        raise Refused('record-binding')
    return value['id']


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--tag', required=True)
    args = parser.parse_args()
    try:
        identifier = created_id(sys.stdin.buffer, args.tag)
    except Exception as error:
        reason = error.reason if isinstance(error, Refused) else 'unknown'
        print('release create response refused: ' + reason, file=sys.stderr)
        return 1
    print(identifier)
    return 0


if __name__ == '__main__':
    raise SystemExit(main())
