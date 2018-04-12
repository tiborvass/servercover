#!/bin/sh

set -e

importpath=github.com/tiborvass/servercover

./clean.sh
go get "$importpath"/cmd/servercover
$GOPATH/bin/servercover -socket cover.sock -o server -coverpkg "$importpath"/example/program/... "$importpath"/example/program/server
./server example.sock &

go build -o client "$importpath"/example/program/client
go test -coverprofile cover.out "$importpath"/example/tests -- "$PWD"/client "$PWD"/example.sock "$PWD"/cover.sock
go tool cover -html=cover.out
