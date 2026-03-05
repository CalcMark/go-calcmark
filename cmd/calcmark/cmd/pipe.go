package cmd

import "os"

// stdinIsPiped returns true when stdin is not a terminal — i.e. data is
// being piped or redirected into the process (e.g. `echo "1+1" | cm`).
func stdinIsPiped() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice == 0
}
