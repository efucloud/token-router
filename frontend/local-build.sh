#!/usr/bin/env bash

set -euo pipefail

GIT_COMMIT=$(git log --pretty=oneline -1 | awk '{print $1}')
echo "Commit: $GIT_COMMIT"

BUILD_DATE=$(date +'%Y/%m/%d %H:%M:%S')
tag="v1.0.0.$(date +'%Y%m%d%H%M')"

echo "Building tag: $tag"
echo "local build start"
npm run build
echo "local build registry.cn-shenzhen.aliyuncs.com/efucloud-dev/token-router-frontend:$tag"
docker buildx build \
  -f Dockerfile.local \
  --build-arg GIT_COMMIT="$GIT_COMMIT" \
  --build-arg BUILD_DATE="$BUILD_DATE" \
  --platform linux/amd64 \
  -t "registry.cn-shenzhen.aliyuncs.com/efucloud-dev/token-router-frontend:$tag" \
  --provenance=false \
  --progress=plain \
  --output=type=registry \
  .
echo "build registry.cn-shenzhen.aliyuncs.com/efucloud-dev/token-router-frontend:$tag success"
