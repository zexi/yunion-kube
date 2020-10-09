#!/bin/bash

set -o errexit
set -o pipefail

if [[ "$DEBUG" == "true" ]]; then
    set -ex ;export PS4='+(${BASH_SOURCE}:${LINENO}): ${FUNCNAME[0]:+${FUNCNAME[0]}(): }'
fi

readlink_mac() {
  cd `dirname $1`
  TARGET_FILE=`basename $1`

  # Iterate down a (possible) chain of symlinks
  while [ -L "$TARGET_FILE" ]
  do
    TARGET_FILE=`readlink $TARGET_FILE`
    cd `dirname $TARGET_FILE`
    TARGET_FILE=`basename $TARGET_FILE`
  done

  # Compute the canonicalized name by finding the physical path
  # for the directory we're in and appending the target file.
  PHYS_DIR=`pwd -P`
  REAL_PATH=$PHYS_DIR/$TARGET_FILE
}

pushd $(cd "$(dirname "$0")"; pwd) > /dev/null
readlink_mac $(basename "$0")
cd "$(dirname "$REAL_PATH")"
CUR_DIR=$(pwd)
SRC_DIR=$(cd .. && pwd)
popd > /dev/null

DOCKER_DIR="$SRC_DIR/"

REGISTRY=${REGISTRY:-docker.io/yunion}
TAG=${TAG:-latest}
PROJ=yunion-kube
image_keyword=kubeserver

build_image() {
    local tag=$1
    local file=$2
    local path=$3
    docker build -t "$tag" -f "$2" "$3"
}

buildx_and_push() {
    local tag=$1
    local file=$2
    local path=$3
    local arch=$4
    local push=()
    docker buildx build -t "$tag" --platform "linux/$arch" -f "$2" "$3" --push
    docker pull "$tag"
}

push_image() {
    local tag=$1
    docker push "$tag"
}

build_process() {
    img_name="$REGISTRY/$image_keyword:$TAG"
    build_image $img_name $DOCKER_DIR/Dockerfile $SRC_DIR
    push_image "$img_name"
}

build_process_with_buildx() {
    local arch=$1

    build_env="GOARCH=$arch "
    img_name="$REGISTRY/$image_keyword:$TAG"
    if [[ $arch == arm64 ]]; then
        img_name="$img_name-$arch"
    fi
	buildx_and_push $img_name $DOCKER_DIR/Dockerfile $SRC_DIR $ARCH
}

cd $SRC_DIR

echo "Start to build for arch[$ARCH]"

case "$ARCH" in
	all)
		for arch in "arm64" "amd64"; do
			build_process_with_buildx $arch
		done
		;;
	arm64 )
		build_process_with_buildx $ARCH
		;;
	*)
		build_process
		;;
esac
