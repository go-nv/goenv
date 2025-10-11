package main

import (
	"github.com/go-nv/goenv/cmd"
)

// Version information (set at build time)
var (
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
)

func main() {
	cmd.SetVersionInfo(version, commit, buildTime)
	cmd.Execute()
}
