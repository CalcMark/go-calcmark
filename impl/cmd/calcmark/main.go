package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	calcmark "github.com/CalcMark/go-calcmark"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "wasm":
		if err := buildWasm(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "version":
		fmt.Printf("calcmark version %s\n", calcmark.Version)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("CalcMark Implementation Tools")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  calcmark wasm [output-dir]    Build WASM module and output with JS glue")
	fmt.Println("  calcmark version              Print version information")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  calcmark wasm                 # Output to current directory")
	fmt.Println("  calcmark wasm ./dist          # Output to ./dist directory")
}

func buildWasm() error {
	// Determine output directory
	outputDir := "."
	if len(os.Args) >= 3 {
		outputDir = os.Args[2]
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Get the module root (where go.mod is)
	moduleRoot, err := findModuleRoot()
	if err != nil {
		return fmt.Errorf("failed to find module root: %w", err)
	}

	wasmDir := filepath.Join(moduleRoot, "impl", "wasm")
	wasmFilename := fmt.Sprintf("calcmark-%s.wasm", calcmark.Version)
	outputWasmPath := filepath.Join(outputDir, wasmFilename)

	fmt.Printf("Building WASM module...\n")
	fmt.Printf("  Version: %s\n", calcmark.Version)
	fmt.Printf("  Source:  %s\n", wasmDir)
	fmt.Printf("  Output:  %s\n", outputWasmPath)

	// Build the WASM module
	cmd := exec.Command("go", "build", "-o", outputWasmPath)
	cmd.Dir = wasmDir
	cmd.Env = append(os.Environ(),
		"GOOS=js",
		"GOARCH=wasm",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to build WASM: %w", err)
	}

	fmt.Println("✓ WASM module built successfully")

	// Copy wasm_exec.js from Go installation
	goRoot, err := getGoRoot()
	if err != nil {
		return fmt.Errorf("failed to determine GOROOT: %w", err)
	}

	wasmExecSrc := filepath.Join(goRoot, "lib", "wasm", "wasm_exec.js")
	wasmExecDst := filepath.Join(outputDir, "wasm_exec.js")

	fmt.Printf("Copying wasm_exec.js...\n")
	fmt.Printf("  From: %s\n", wasmExecSrc)
	fmt.Printf("  To:   %s\n", wasmExecDst)

	if err := copyFile(wasmExecSrc, wasmExecDst); err != nil {
		return fmt.Errorf("failed to copy wasm_exec.js: %w", err)
	}

	fmt.Println("✓ wasm_exec.js copied successfully")
	fmt.Println()
	fmt.Println("WASM build complete!")
	fmt.Printf("  Output files:\n")
	fmt.Printf("    %s\n", outputWasmPath)
	fmt.Printf("    %s\n", wasmExecDst)

	return nil
}

func findModuleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found")
		}
		dir = parent
	}
}

func getGoRoot() (string, error) {
	cmd := exec.Command("go", "env", "GOROOT")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", err
	}

	return strings.TrimSpace(out.String()), nil
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	return destFile.Sync()
}
