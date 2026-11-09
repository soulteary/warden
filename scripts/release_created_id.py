#!/usr/bin/env python3
"""Authorize only an explicit HTTP 201 create response, never an existing ID."""
import argparse
import json
import re
import sys

LIMIT = 1024 * 1024


def created_id(stream, tag):
    raw = stream.read(LIMIT + 1)
    if len(raw) > LIMIT:
        raise ValueError('response budget')
    # gh --include may render LF headers; raw HTTP CRLF is also accepted.
    if b'\r\n\r\n' in raw:
        header, body = raw.split(b'\r\n\r\n', 1)
        if b'\r' in header.replace(b'\r\n', b''):
            raise ValueError('header framing')
        lines = header.split(b'\r\n')
    elif b'\n\n' in raw and b'\r' not in raw:
        header, body = raw.split(b'\n\n', 1)
        lines = header.split(b'\n')
    else:
        raise ValueError('header framing')
    if len(header) > 32768 or len(lines) > 129:
        raise ValueError('header budget')
    if not re.fullmatch(rb'HTTP/(?:1\.[01]|2(?:\.0)?) 201(?: Created)?', lines[0]):
        raise ValueError('not explicitly created')
    content_types = []
    for line in lines[1:]:
        match = re.fullmatch(rb'([!#$%&\x27*+.^_`|~0-9A-Za-z-]+):[ \t]*([\x20-\x7e\t]*)', line)
        if not match:
            raise ValueError('header syntax')
        if match[1].lower() == b'content-type':
            content_types.append(match[2].lower().strip())
    if len(content_types) != 1 or content_types[0].split(b';', 1)[0].strip() != b'application/json':
        raise ValueError('response media type')
    def unique_object(pairs):
        result = {}
        for key, item in pairs:
            if key in result:
                raise ValueError('duplicate JSON key')
            result[key] = item
        return result
    def reject_constant(value):
        raise ValueError('non-JSON constant')
    value = json.loads(body, object_pairs_hook=unique_object, parse_constant=reject_constant)
    if not isinstance(value, dict) or type(value.get('id')) is not int or value['id'] <= 0:
        raise ValueError('new release ID')
    if value.get('tag_name') != tag or value.get('draft') is not True:
        raise ValueError('new release binding')
    return value['id']


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--tag', required=True)
    args = parser.parse_args()
    try:
        identifier = created_id(sys.stdin.buffer, args.tag)
    except Exception:
        print('release create response refused', file=sys.stderr)
        return 1
    print(identifier)
    return 0


if __name__ == '__main__':
    raise SystemExit(main())
