package main

import (
	"flag"
	"testing"

	"github.com/tiborvass/maincover"
)

var testBinary, serverSocket, maincoverSocket string

func init() {
	flag.Parse()
	testBinary = flag.Args()[0]
	serverSocket = flag.Args()[1]
	maincoverSocket = flag.Args()[2]
}

type Test struct {
	in  int
	out string
}

var tests = []Test{
	{-1, "negative"},
	{5, "small"},
}

func TestMain(m *testing.M) {
	flag.Parse()
	maincover.TestMain(m, "unix", maincoverSocket)
}

func TestSize(t *testing.T) {
	for i, test := range tests {
		size, err := CmdSize(test.in)
		if err != nil {
			t.Fatal(err)
		}
		if size != test.out {
			t.Errorf("#%d: Size(%d)=%s; want %s", i, test.in, size, test.out)
		}
	}
}
