package main

import (
	"github.com/CalcMark/go-calcmark/cmd/calcmark/cmd"
)

// Version info set via ldflags at build time
var (
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	// Forward version info to cmd package
	cmd.Version = Version
	cmd.BuildTime = BuildTime

	cmd.Execute()
}
