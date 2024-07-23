#!/bin/bash

if [ "$IMG" == "" ]; then
  IMG="ttl.sh/$(uuidgen):2h"
fi

# install customize if not installed
make kustomize

function buildControllerImage() {
  echo "Build operator controller container image ${IMG}" >&2
  make docker-build IMG="$IMG" && make docker-push IMG="$IMG"
  return $?
}

# $1 out file
function generateOperatorResources() {
  echo "Generating resources..." >&2
  make generate-operator-resources IMG="$IMG" OUT_FILE="$1"
}

function deployOperator() {
  echo "Deploy to cluster" >&2
  make deploy IMG="$IMG"
}

case "$1" in
  "easy")
    echo "Installing cert-manager" >&2
    kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.15.1/cert-manager.yaml
    echo "Building operator" >&2
    docker build -t "${IMG}" --build-arg IMG="${IMG}" . && docker push "${IMG}"
    echo "Installing operator" >&2
    docker run "${IMG}" /print-resources | kubectl apply -f -
  ;;
  "build_and_deploy")
    buildControllerImage && deployOperator || exit 1
  ;;
  "build_and_generate")
    OUTF="operator-resources.yaml"
    buildControllerImage && generateOperatorResources "$OUTF" || exit 1
    echo "Operator resources saved in $OUTF" >&2
  ;;
  *)
    echo -e "Use $0 with one of arguments\n
      easy - build everything and install on your cluster
      build_and_generate - build, push controller container and generate operator resources
      build_and_deploy - build, push controller container and deploy operator
    "
    exit 3
esac
