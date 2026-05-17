package main

import (
	"encoding/json"
	"fmt"
	"os"
)

var (
	release   = "v0.0.1"
	buildDate = "2026-05-04T00:00:00Z"
	gitHash   = "dev"
)

func printVersion() {
	if err := json.NewEncoder(os.Stdout).Encode(struct {
		Release   string
		BuildDate string
		GitHash   string
	}{
		Release:   release,
		BuildDate: buildDate,
		GitHash:   gitHash,
	}); err != nil {
		fmt.Printf("error while decode version info: %v\n", err)
	}
}
