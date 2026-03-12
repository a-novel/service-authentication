#!/bin/bash

set -e

case "$OSTYPE" in
  darwin*|bsd*)
    echo "Using BSD sed style"
    sed -i '' -E "s|($1)v[0-9.]+|\1v$(node -p -e "require('./package.json').version")|g" $2
    ;;
  *)
    echo "Using GNU sed style"
    sed -i -E "s|\($1\)v[0-9.]+|\1v$(node -p -e "require('./package.json').version")|g" $2
    ;;
esac
