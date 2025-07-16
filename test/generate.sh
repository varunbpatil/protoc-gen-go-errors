#!/bin/bash
set -e

pushd .. > /dev/null
go build -o bin/protoc-gen-go-errors ./.
popd > /dev/null

protoc \
  -I=. \
  -I=../proto \
  --go_out=. \
  --go-errors_out=. \
  --plugin=protoc-gen-go-errors=../bin/protoc-gen-go-errors \
  errors.proto
