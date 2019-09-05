#!/usr/bin/env bash

# exit when any command fails
set -e

DIR=$(dirname $0)

echo "<> Downloading source and updating dependecies"
go get -v -t -d ./...
echo

echo "<> Running go test"
go test -v ./...
echo

echo "<> Generating build artifacts"
source $DIR/all.sh
echo

echo "<> Generating index.html"
node $DIR/www.js
