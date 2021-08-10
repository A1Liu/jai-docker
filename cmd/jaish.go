package main

import (
	"context"
	"os"
	"strings"

	jai "a1liu.com/jai-docker"
)

func main() {
	argstring := strings.Join(os.Args[1:], " ")
	commandStatus := jai.RunCmd(context.Background(), "/bin/bash", []string{"-c", argstring})
	if commandStatus != 0 {
		os.Exit(commandStatus)
	}

	os.Exit(0)
}
