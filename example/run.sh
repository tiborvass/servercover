#!/bin/sh

set -e

importpath=github.com/tiborvass/maincover

./clean.sh
go get "$importpath"/cmd/maincover
$GOPATH/bin/maincover -socket maincover.sock -o server -coverpkg "$importpath"/example/program/... "$importpath"/example/program/server
./server example.sock &

go build -o client "$importpath"/example/program/client
go test -coverprofile cover.out "$importpath"/example/tests -- "$PWD"/client "$PWD"/example.sock "$PWD"/maincover.sock
go tool cover -html=cover.out
