#!/bin/bash
# Renders every .mjml source in the tree to a sibling .html, which is what the mails
# package embeds. Run it after editing any template.

set -e

for i in `find . -name "*.mjml" -type f`
do
  pnpm mjml $i --config.beautify false --config.minify false -o ${i%.*}.html
done
