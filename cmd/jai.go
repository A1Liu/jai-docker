package main

import (
	"context"
	"os"

	jai "a1liu.com/jai-docker"
)

func main() {
	commandStatus := jai.RunCmd(context.Background(),
		"/root/jai-docker/jai/bin/jai-linux", os.Args[1:])
	if commandStatus != 0 {
		os.Exit(commandStatus)
	}

	os.Exit(0)
}
