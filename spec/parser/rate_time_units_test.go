package parser

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/ast"
)

func TestRateWithTimeUnitSynonyms(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		expectedPer string
	}{
		// Second variants
		{name: "s", input: "100 MB/s\n", expectError: false, expectedPer: "s"},
		{name: "sec", input: "100 MB/sec\n", expectError: false, expectedPer: "sec"},
		{name: "second", input: "100 MB per second\n", expectError: false, expectedPer: "second"},
		{name: "seconds", input: "100 MB per seconds\n", expectError: false, expectedPer: "seconds"},

		// Minute variants
		{name: "m", input: "1k req/m\n", expectError: false, expectedPer: "m"},
		{name: "min", input: "1k req/min\n", expectError: false, expectedPer: "min"},
		{name: "minute", input: "1k req per minute\n", expectError: false, expectedPer: "minute"},
		{name: "minutes", input: "1k req per minutes\n", expectError: false, expectedPer: "minutes"},

		// Hour variants
		{name: "h", input: "$10/h\n", expectError: false, expectedPer: "h"},
		{name: "hr", input: "$10/hr\n", expectError: false, expectedPer: "hr"},
		{name: "hour", input: "$10 per hour\n", expectError: false, expectedPer: "hour"},
		{name: "hours", input: "$10 per hours\n", expectError: false, expectedPer: "hours"},

		// Day variants
		{name: "d", input: "5 GB/d\n", expectError: false, expectedPer: "d"},
		{name: "day", input: "5 GB per day\n", expectError: false, expectedPer: "day"},
		{name: "days", input: "5 GB per days\n", expectError: false, expectedPer: "days"},

		// Week variants
		{name: "w", input: "100 hours/w\n", expectError: false, expectedPer: "w"},
		{name: "wk", input: "100 hours/wk\n", expectError: false, expectedPer: "wk"},
		{name: "week", input: "100 hours per week\n", expectError: false, expectedPer: "week"},
		{name: "weeks", input: "100 hours per weeks\n", expectError: false, expectedPer: "weeks"},

		// Month variants
		{name: "mo", input: "$50/mo\n", expectError: false, expectedPer: "mo"},
		{name: "month", input: "$50 per month\n", expectError: false, expectedPer: "month"},
		{name: "months", input: "$50 per months\n", expectError: false, expectedPer: "months"},

		// Year variants
		{name: "y", input: "1M users/y\n", expectError: false, expectedPer: "y"},
		{name: "yr", input: "1M users/yr\n", expectError: false, expectedPer: "yr"},
		{name: "year", input: "1M users per year\n", expectError: false, expectedPer: "year"},
		{name: "years", input: "1M users per years\n", expectError: false, expectedPer: "years"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := Parse(tt.input)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
				return
			}

			if !tt.expectError && err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if !tt.expectError {
				if len(nodes) == 0 {
					t.Fatal("No nodes returned")
				}

				rate, ok := nodes[0].(*ast.RateLiteral)
				if !ok {
					t.Fatalf("Expected RateLiteral, got %T", nodes[0])
				}

				if rate.PerUnit != tt.expectedPer {
					t.Errorf("Expected PerUnit '%s', got '%s'", tt.expectedPer, rate.PerUnit)
				}

				t.Logf("✓ Parsed: %s", rate.String())
			}
		})
	}
}
