package main

import (
	"context"
	"io"
	"net"
	"net/http"
	"os"
)

func main() {
	unixSocket := os.Args[1]
	httpc := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", unixSocket)
			},
		},
	}
	resp, err := httpc.Get("http://unix/size?param=" + os.Args[2])
	if err != nil {
		panic(err)
	}
	io.Copy(os.Stdout, resp.Body)
}
