//go:build !js && !wasm

package store

import (
	"bytes"
	"os/exec"
)

// RealExecutor wraps os/exec for production use.
type RealExecutor struct{}

// Run executes a command with captured stdout/stderr.
func (RealExecutor) Run(name string, args ...string) (stdout, stderr []byte, err error) {
	cmd := exec.Command(name, args...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err = cmd.Run()
	return outBuf.Bytes(), errBuf.Bytes(), err
}

// LookPath searches for an executable on PATH.
func (RealExecutor) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}
