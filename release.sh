#! /bin/bash
set -ex

NAME=$(basename `git rev-parse --show-toplevel`)
ARCH=$(uname -m)

rm -rf release
mkdir release

declare -a PLATFORMS=("Linux" "Darwin")
declare -a FILES=()

for PLATFORM in ${PLATFORMS[@]}; do
  if [ -d "./build/$PLATFORM" ]; then
    echo "Compressing the ${PLATFORM} relevant binary ..."
    FILE="${NAME}_${VERSION}_${PLATFORM}_${ARCH}.tgz"
    LATEST_FILE="${NAME}_latest_${PLATFORM}_${ARCH}.tgz"
    FILES+=("$FILE")
    tar -zcf "release/${FILE}" -C build/$PLATFORM $BINARY
    cp ./release/$FILE ./release/$LATEST_FILE
  fi
done

if (( ${#FILES[@]} )); then
  echo "Creating release ${VERSION} ..."
else
  echo "No file found to release."
  exit 0
fi

OUTPUT=$(gh release list | grep "^${VERSION}" | true)
if [ -z "$OUTPUT" ]; then 
  
  printf -v RELEASABLE_FILES './release/%s ' "${FILES[@]}"
  gh release create "v${VERSION}" $RELEASABLE_FILES -t ${VERSION} -n ""

  FILE_NAME="Makefile"
  SEARCH=${VERSION}
  REPLACE=${VERSION%.*}.$((${VERSION#*.}+1))

  if [[ $SEARCH != "" && $REPLACE != "" ]]; then
    echo "Increasing version from ${SEARCH} to ${REPLACE} in the ${FILE_NAME}"
    SEARCH_TEXT="export VERSION=${SEARCH}"
    REPLACE_TEXT="export VERSION=${REPLACE}"
    sed -i "s/$SEARCH_TEXT/$REPLACE_TEXT/" $FILE_NAME
  fi
else
  echo "The release v${VERSION} already exists on the github."
fi

