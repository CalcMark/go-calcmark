package cmd

import (
	"bytes"
	"os/exec"
	"testing"
)

func TestCompletion_BashWorks(t *testing.T) {
	binary := buildCM(t)

	cmd := exec.Command(binary, "completion", "bash")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("cm completion bash failed: %v\nstderr: %s", err, stderr.String())
	}

	if !bytes.Contains(stdout.Bytes(), []byte("bash completion")) {
		t.Errorf("expected bash completion output, got %q", stdout.String()[:100])
	}
}

func TestCompletion_PluralAliasWorks(t *testing.T) {
	binary := buildCM(t)

	cmd := exec.Command(binary, "completions", "bash")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("cm completions bash failed: %v\nstderr: %s", err, stderr.String())
	}

	if !bytes.Contains(stdout.Bytes(), []byte("bash completion")) {
		t.Errorf("expected bash completion output via plural alias, got %q", stdout.String()[:100])
	}
}

func TestCompletion_NoArgShowsError(t *testing.T) {
	binary := buildCM(t)

	cmd := exec.Command(binary, "completion")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected error when no shell argument provided")
	}
}
