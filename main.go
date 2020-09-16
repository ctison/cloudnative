package main

import (
	"fmt"
	"os"

	"github.com/ctison/cloudnative/pkg/cli"
)

var version = "v0.0.0"

func main() {
	if err := cli.New(version).Run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
