package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/tiborvass/maincover/example/program/subprogram"
)

func main() {
	if len(os.Args) == 1 {
		return
	}
	a, err := strconv.Atoi(os.Args[1])
	if err != nil {
		return
	}
	fmt.Println(subprogram.Size(a))
}
