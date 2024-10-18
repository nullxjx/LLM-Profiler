#!/usr/bin/env bash

commit_id="$(git rev-parse HEAD)"
registry="docker.io"
repository="thexjx/llm-perf-analyzer"
tag="${commit_id}"

function build_perf_analyzer() {
    image_name="${registry}/${repository}:${tag}"
    echo image_name: "${image_name}"
    docker build --no-cache --platform linux/amd64 -t "${image_name}" -f build/Dockerfile .
    docker push "${image_name}"
}

build_perf_analyzer