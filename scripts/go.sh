#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
if [ -x .tools/go/bin/go ]; then
  PATH="$PWD/.tools/go/bin:$PATH"
  export PATH
fi
if ! command -v go >/dev/null 2>&1; then
  printf '%s\n' 'Go 1.27.3+ is required. Install it or extract the official SDK into .tools/go.' >&2
  exit 1
fi
exec go "$@"
