package main

import (
	"context"
	"os"
	"strings"

	jai "a1liu.com/jai-docker"
)

func main() {

	args := make([]string, len(os.Args))
	copy(args[1:], os.Args[1:])
	args[0] = "-c"

	argstring := strings.Join(args, " ")

	commandStatus := jai.RunCmd(context.Background(), "/bin/bash", []string{argstring})
	if commandStatus != 0 {
		os.Exit(commandStatus)
	}

	os.Exit(0)
}
