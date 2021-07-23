#!/bin/bash

PUBLIC_DIR=../../../public

GOOS=js GOARCH=wasm go build -o $PUBLIC_DIR/parsing.wasm
cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" $PUBLIC_DIR/go_wasm_exec.js
