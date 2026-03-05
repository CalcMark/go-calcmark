package cmd

import (
	"os"

	"github.com/mattn/go-isatty"
)

// stdinIsPiped returns true when stdin is not a terminal — i.e. data is
// being piped or redirected into the process (e.g. `echo "1+1" | cm`).
//
// Uses go-isatty for reliable cross-platform detection. Falls back to
// os.Stat mode check if the fd-based check is inconclusive.
func stdinIsPiped() bool {
	fd := os.Stdin.Fd()
	if isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd) {
		return false
	}
	// fd is not a terminal — could be a pipe, redirect, or /dev/null.
	// Check whether there is actual data available by examining the mode.
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	mode := fi.Mode()
	// Named pipe (shell pipe) or regular file (redirect) — treat as piped.
	// Character devices like /dev/null are NOT treated as pipes.
	return mode&os.ModeNamedPipe != 0 || mode.IsRegular()
}
