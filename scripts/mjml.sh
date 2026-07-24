#!/bin/bash
# Renders every .mjml source in the tree to a sibling .html, which is what the mails
# package embeds. Run it after editing any template.

set -e

# NUL-delimited, so a template path containing a space or newline survives the split. Read through
# process substitution rather than a pipe, so the loop stays in this shell and `set -e` still sees a
# failed render instead of it being swallowed by a subshell.
while IFS= read -r -d '' i; do
  pnpm mjml "$i" --config.beautify false --config.minify false -o "${i%.*}.html"
done < <(find . -name "*.mjml" -type f -print0)
