#!/usr/bin/env bash
set -e

NAME=Brucheion
RELEASE=release
VERSION=$(git describe --abbrev=0 --tags)
LDFLAGS="-X main.BuildTime=$(date +%FT%T%z) -X main.Version=${VERSION}"

# iterate over tuples <https://stackoverflow.com/a/9713142>
OLDIFS=$IFS; IFS=',';

mkdir -p "$RELEASE"

for i in darwin,amd64 windows,386 windows,amd64; do
  set -- $i
  GOOS=$1
  GOARCH=$2
  echo "building ${GOOS} ${GOARCH}"

  [[ $GOOS = "windows" ]] && BIN="${NAME}.exe" || BIN="${NAME}"

	env GOOS="$GOOS" GOARCH="$GOARCH" go build -ldflags "${LDFLAGS}" -o "${RELEASE}/${BIN}"
	(cd $RELEASE &&
	  tar czvf "${NAME}_${VERSION}_${GOOS}_${GOARCH}.tar.gz" "$BIN" &&
	  rm "$BIN")
done

IFS=$OLDIFS
