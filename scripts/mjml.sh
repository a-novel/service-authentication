#!/bin/bash

set -e

for i in `find . -name "*.mjml" -type f`
do
  pnpm mjml $i --config.beautify false --config.minify false -o ${i%.*}.html
done
