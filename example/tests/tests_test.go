package main

import (
	"testing"

	"github.com/tiborvass/maincover"
)

type Test struct {
	in  int
	out string
}

var tests = []Test{
	{-1, "negative"},
	{5, "small"},
}

func TestMain(m *testing.M) {
	//flag.Parse()
	maincover.TestMain(m, "unix", "/tmp/maincover.sock")
}

func TestSize(t *testing.T) {
	for i, test := range tests {
		size, err := Size(test.in)
		if err != nil {
			t.Fatal(err)
		}
		if size != test.out {
			t.Errorf("#%d: Size(%d)=%s; want %s", i, test.in, size, test.out)
		}
	}
}
