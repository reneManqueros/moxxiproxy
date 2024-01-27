package main

import (
	"fmt"
	"log"
	"moxxiproxy/cmd"
	"runtime/debug"
)

func main() {
	log.Println(CommitInfo())
	cmd.Execute()
}

func CommitInfo() string {
	revision := ""
	buildTime := ""

	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				revision = setting.Value
			}
			if setting.Key == "vcs.time" {
				buildTime = setting.Value
			}
		}
	}

	return fmt.Sprintf("Rev: %s @ %s", revision, buildTime)
}
