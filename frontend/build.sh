#!/usr/bin/env bash

set -euo pipefail

for lockfile in yarn.lock pnpm-lock.yaml; do
  if [[ -e "$lockfile" ]]; then
    echo "Error: found $lockfile; this project uses npm and package-lock.json only." >&2
    exit 1
  fi
done

GIT_COMMIT=$(git log --pretty=oneline -1 | awk '{print $1}')
echo "Commit: $GIT_COMMIT"

BUILD_DATE=$(date +'%Y/%m/%d %H:%M:%S')
tag="v1.0.0.$(date +'%Y%m%d%H%M')"

echo "Building tag: $tag"

IMAGE="ghcr.io/efucloud/token-router-frontend:$tag"

docker buildx build \
  --build-arg GIT_COMMIT="$GIT_COMMIT" \
  --build-arg BUILD_DATE="$BUILD_DATE" \
  --platform linux/amd64,linux/arm64 \
  -t "$IMAGE" \
  --provenance=false \
  --progress=plain \
  --output=type=registry \
  .

echo "Push success: $IMAGE"
