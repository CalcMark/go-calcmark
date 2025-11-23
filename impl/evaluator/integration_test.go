package evaluator_test

import (
	"testing"

	"github.com/CalcMark/go-calcmark/impl/evaluator"
)

// TestEvaluatorWithNewParser tests that the evaluator works with the new gocc parser
func TestEvaluatorWithNewParser(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "simple addition",
			input:   "2 + 3\n",
			want:    "5",
			wantErr: false,
		},
		{
			name:    "assignment and use",
			input:   "x = 5\nx + 10\n",
			want:    "15",
			wantErr: false,
		},
		{
			name:    "multiple statements",
			input:   "a = 10\nb = 20\na + b\n",
			want:    "30",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := evaluator.NewContext()
			results, err := evaluator.Evaluate(tt.input, ctx)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Evaluate() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Evaluate() unexpected error = %v", err)
				return
			}

			if len(results) == 0 {
				t.Errorf("Evaluate() returned no results")
				return
			}

			// Check last result
			got := results[len(results)-1].String()
			if got != tt.want {
				t.Errorf("Evaluate() = %v, want %v", got, tt.want)
			}
		})
	}
}
