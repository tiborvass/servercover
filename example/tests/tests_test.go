package main

import (
	"os"
	"testing"

	"github.com/tiborvass/servercover"
	"github.com/tiborvass/servercover/example/program/client"
)

var serverSocket = os.Getenv("EXAMPLE_SOCK")

type Test struct {
	in  string
	out string
}

var tests = []Test{
	{"-1", "negative"},
	{"5", "small"},
}

func TestMain(m *testing.M) {
	servercover.TestMain(m)
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
