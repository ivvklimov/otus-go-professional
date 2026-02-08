package main

import (
	"os"
)

func main() {
	if len(os.Args) < 3 {
		os.Exit(1)
	}

	dir := os.Args[1]
	command := os.Args[2:]

	env, err := ReadDir(dir)
	if err != nil {
		os.Exit(1)
	}

	returnCode := RunCmd(command, env)
	os.Exit(returnCode)
}
