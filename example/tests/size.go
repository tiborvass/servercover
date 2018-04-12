package main

import (
	"fmt"
	"os/exec"
	"strings"
)

func CmdSize(a int) (string, error) {
	cmd := exec.Command(testBinary, serverSocket, fmt.Sprintf("%d", a))
	fmt.Println("tibor", cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%d: %v: %s", a, err, out)
	}
	return strings.TrimSpace(string(out)), nil
}
