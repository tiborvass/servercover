package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"

	"github.com/tiborvass/servercover/example/program/subprogram"

	_ "net/rpc"
	_ "testing"
)

func atoi(s string) int {
	a, err := strconv.Atoi(s)
	if err != nil {
		panic(err)
	}
	return a
}

func main() {
	http.HandleFunc("/size", func(w http.ResponseWriter, r *http.Request) {
		a := r.FormValue("param")
		i := atoi(a)
		fmt.Fprintln(w, subprogram.Size(i))
	})
	os.Remove("/tmp/example.sock")
	l, err := net.Listen("unix", os.Args[1])
	if err != nil {
		panic(err)
	}
	log.Fatal(http.Serve(l, nil))
}
