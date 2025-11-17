package main

import (
	"fmt"
	"os"

	calcmark "github.com/CalcMark/go-calcmark"
	"github.com/CalcMark/go-calcmark/spec/grammar"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version":
			fmt.Println(calcmark.Version)
			return
		case "help", "-h", "--help":
			printUsage()
			return
		default:
			fmt.Fprintf(os.Stderr, "Unknown argument: %s\n\n", os.Args[1])
			printUsage()
			os.Exit(1)
		}
	}

	// Default: output EBNF grammar
	printEBNF()
}

func printUsage() {
	fmt.Println("CalcMark Specification Tools")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  cmspec              Generate EBNF grammar")
	fmt.Println("  cmspec version      Print version information")
	fmt.Println("  cmspec help         Show this help message")
	fmt.Println()
	fmt.Println("Example:")
	fmt.Println("  cmspec > calcmark.ebnf")
}

func printEBNF() {
	ebnf := grammar.GenerateEBNF(calcmark.Version)
	fmt.Print(ebnf)
}
