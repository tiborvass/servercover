package main

import (
	"fmt"
	"os/exec"
	"strings"
)

const testBinary = "/Users/tiborvass/go/src/github.com/tiborvass/maincover/example/program/program"

func Size(a int) (string, error) {
	cmd := exec.Command(testBinary, fmt.Sprintf("%d", a))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%d: %v: %s", a, err, out)
	}
	return strings.TrimSpace(string(out)), nil
}
