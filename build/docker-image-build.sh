#!/bin/bash
VERSION=$(git describe --tags --abbrev=0)
COMMIT=$(git rev-parse --short HEAD)

echo "最近版本；$VERSION / $COMMIT"
echo "$VERSION">RELEASE_VERSION
echo "$COMMIT">RELEASE_COMMIT

docker build -t warden-release -f docker/Dockerfile .

# rm RELEASE_VERSION
# rm RELEASE_COMMIT
