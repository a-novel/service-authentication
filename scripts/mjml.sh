#!/bin/bash
# Renders localized email sources to the HTML embedded by the mails package.
# Pass --validate to check sources without writing generated files.

set -e

if [ "$#" -gt 1 ] || { [ "$#" -eq 1 ] && [ "$1" != "--validate" ]; }; then
  printf 'Usage: %s [--validate]\n' "$0" >&2
  exit 1
fi

mail_root="$(cd "$(dirname "$0")/../internal/models/mails" && pwd)"

for source in "$mail_root"/*/*.mjml; do
  output="${source%.*}.html"
  if [ "${1:-}" = "--validate" ]; then
    output=/dev/null
  fi

  pnpm mjml "$source" \
    --config.allowIncludes true \
    --config.includePath "$mail_root" \
    --config.validationLevel strict \
    --config.beautify true \
    --config.minify false \
    -o "$output"
done
