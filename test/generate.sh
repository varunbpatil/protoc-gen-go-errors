#!/bin/bash
set -e

pushd .. > /dev/null
protoc --go_out=. --go_opt=paths=source_relative -I=. errors/options.proto
go build -o bin/protoc-gen-go-errors ./.
popd > /dev/null

protoc \
  -I=. \
  -I=.. \
  --go_out=. \
  --go-errors_out=. \
  --plugin=protoc-gen-go-errors=../bin/protoc-gen-go-errors \
  errors.proto
