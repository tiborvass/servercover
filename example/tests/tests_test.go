package main

import (
	"flag"
	"testing"

	"github.com/tiborvass/servercover"
	"github.com/tiborvass/servercover/example/program/client"
)

var serverSocket string

var coverAddr = flag.String("cover.addr", "", "Address to servercover")

func init() {
	flag.Parse()
	serverSocket = flag.Args()[0]
}

type Test struct {
	in  string
	out string
}

var tests = []Test{
	{"-1", "negative"},
	{"5", "small"},
}

func TestMain(m *testing.M) {
	flag.Parse()
	if *coverAddr == "" {
		panic("-cover.addr is needed")
	}
	servercover.TestMain(m, "unix", *coverAddr)
}

func TestSize(t *testing.T) {
	for i, test := range tests {
		size, err := client.Size(serverSocket, test.in)
		if err != nil {
			t.Fatal(err)
		}
		if size != test.out {
			t.Errorf("#%d: Size(%s)=%s; want %s", i, test.in, size, test.out)
		}
	}
}
