package client

import (
	"bufio"
	"context"
	"net"
	"net/http"
)

func Size(unixSocket string, param string) (string, error) {
	httpc := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", unixSocket)
			},
		},
	}
	resp, err := httpc.Get("http://unix/size?param=" + param)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	s := bufio.NewScanner(resp.Body)
	if s.Scan() {
		return s.Text(), s.Err()
	}
	return "", s.Err()
}
