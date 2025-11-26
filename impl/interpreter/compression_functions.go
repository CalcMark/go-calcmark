package interpreter

import (
	"fmt"
	"strings"

	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/shopspring/decimal"
)

// Compression ratios for typical text/log data
// Ratio represents: original_size / compressed_size
var compressionRatios = map[string]float64{
	"gzip":   3.0, // 3:1 compression (33% of original)
	"lz4":    2.0, // 2:1 compression (50% of original, fast)
	"zstd":   3.5, // 3.5:1 compression (28% of original)
	"bzip2":  4.0, // 4:1 compression (25% of original, slow)
	"none":   1.0, // 1:1 no compression (100% of original)
	"snappy": 2.5, // 2.5:1 compression (40% of original, fast)
}

// calculateCompression estimates the compressed size of data.
// Time Complexity: O(1) - map lookup + arithmetic
//
// Examples:
//   - compress(1 GB, gzip) → ~341 MB (1024/3 ≈ 341)
//   - compress(100 MB, lz4) → 50 MB (100/2)
//   - compress(500 MB, none) → 500 MB (no compression)
func calculateCompression(size *types.Quantity, compressionType string) (*types.Quantity, error) {
	typeLower := strings.ToLower(strings.TrimSpace(compressionType))

	// Look up compression ratio
	ratio, exists := compressionRatios[typeLower]
	if !exists {
		return nil, fmt.Errorf(
			"unknown compression type '%s' (valid types: gzip, lz4, zstd, bzip2, snappy, none)",
			compressionType,
		)
	}

	// Calculate compressed size: original / ratio
	compressedValue := size.Value.Div(decimal.NewFromFloat(ratio))

	// Return quantity with same unit as input
	return &types.Quantity{
		Value: compressedValue,
		Unit:  size.Unit,
	}, nil
}
