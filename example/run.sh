#!/bin/sh

set -e

pushd $(dirname "$0") >/dev/null
trap 'popd >/dev/null' EXIT

importpath=github.com/tiborvass/servercover

./clean.sh
go get "$importpath"/cmd/servercover
$GOPATH/bin/servercover -socket cover.sock -o server -coverpkg "$importpath"/example/program/... "$importpath"/example/program/server
go build -o client "$importpath"/example/program/client

./server example.sock &
go test -coverprofile cover.out "$importpath"/example/tests -- "$PWD"/client "$PWD"/example.sock "$PWD"/cover.sock
kill -9 %1

go tool cover -func=cover.out
# HTML output works too
#go tool cover -html=cover.out

echo
echo "You may clean up with clean.sh"
