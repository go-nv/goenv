package main

import (
	"github.com/go-nv/goenv/cmd"

	// Import subpackages to trigger their init() functions which register commands
	_ "github.com/go-nv/goenv/cmd/aliases"
	_ "github.com/go-nv/goenv/cmd/compliance"
	_ "github.com/go-nv/goenv/cmd/core"
	_ "github.com/go-nv/goenv/cmd/diagnostics"
	_ "github.com/go-nv/goenv/cmd/hooks"
	_ "github.com/go-nv/goenv/cmd/integrations"
	_ "github.com/go-nv/goenv/cmd/legacy"
	_ "github.com/go-nv/goenv/cmd/meta"
	_ "github.com/go-nv/goenv/cmd/shell"
	_ "github.com/go-nv/goenv/cmd/shims"
	_ "github.com/go-nv/goenv/cmd/tools"
	_ "github.com/go-nv/goenv/cmd/version"
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
