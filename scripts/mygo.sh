#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
if [ -x .tools/go/bin/go ]; then
  PATH="$PWD/.tools/go/bin:$PATH"
  export PATH
fi
if ! command -v go >/dev/null 2>&1; then
  printf '%s\n' 'Go 1.27.3+ is required. See README.md.' >&2
  exit 1
fi
# LocalOps uses C/Objective-C system APIs on macOS. Other targets use pure Go.
case "$(go env GOOS)" in
  darwin) export CGO_ENABLED=1 ;;
  *) export CGO_ENABLED=0 ;;
esac
exec ./node_modules/.bin/mygo "$@"
