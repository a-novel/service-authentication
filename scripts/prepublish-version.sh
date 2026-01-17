#!/bin/bash

set -e

case "$OSTYPE" in
  darwin*|bsd*)
    echo "Using BSD sed style"
    sed_no_backup=( -i '' )
    ;;
  *)
    echo "Using GNU sed style"
    sed_no_backup=( -i )
    ;;
esac

sed ${sed_no_backup[@]} -E "s|($1)v[0-9.]+|\1v$(node -p -e "require('./package.json').version")|g" $2
