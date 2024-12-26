#!/usr/bin/env bash

commit_id="$(git rev-parse HEAD)"
registry="docker.io"
tag="${commit_id}"

function build() {
    repository="triton/llm_profiler"
    image_name="${registry}/${repository}:${tag}"
    echo image_name: "${image_name}"
    docker build --no-cache --platform linux/amd64 -t "${image_name}" -f build/Dockerfile .
    docker push "${image_name}"
}

build