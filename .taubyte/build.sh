#!/bin/bash

# wasm.sh expects CODE to point at the repo root (see /utils/wasm.sh).
export CODE="${CODE:-/src}"

. /utils/wasm.sh

build "${FILENAME}"
ret=$?
echo -n $ret > /out/ret-code
exit $ret
