#!/bin/bash

set -e

pnpm version $1 --workspaces --workspaces-update=false --no-git-tag-version
pnpm prepublish:doc

git add -A
git commit -m "$(node -p "require('./package.json').version")"
git tag v$(node -p "require('./package.json').version")
git push
git push --tags
