// Package filecheck validates that file content is safe to interpret as CalcMark.
// It provides defense-in-depth against binary files renamed with .cm extensions.
package filecheck

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// IsCalcMarkExtension reports whether the file at path has a recognized
// CalcMark extension (.cm or .calcmark, case-insensitive).
func IsCalcMarkExtension(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".cm" || ext == ".calcmark"
}

// binarySignature maps a human-readable format name to its magic bytes.
type binarySignature struct {
	name  string
	magic []byte
}

// knownBinarySignatures lists magic-byte prefixes for common binary formats.
// Checked in order; the first match wins.
var knownBinarySignatures = []binarySignature{
	{"PNG", []byte{0x89, 0x50, 0x4E, 0x47}},
	{"JPEG", []byte{0xFF, 0xD8, 0xFF}},
	{"GIF", []byte("GIF8")},
	{"PDF", []byte("%PDF")},
	{"ZIP", []byte("PK\x03\x04")},
	{"GZIP", []byte{0x1F, 0x8B}},
	{"ELF", []byte{0x7F, 0x45, 0x4C, 0x46}},
	{"PE/MZ", []byte("MZ")},
	{"WASM", []byte{0x00, 0x61, 0x73, 0x6D}},
	{"RIFF", []byte("RIFF")},                         // WAV, AVI, WebP
	{"OGG", []byte("OggS")},                          // Ogg Vorbis
	{"FLAC", []byte("fLaC")},                         // FLAC audio
	{"BMP", []byte("BM")},                            // Bitmap image
	{"TIFF-LE", []byte{0x49, 0x49, 0x2A, 0x00}},      // TIFF little-endian
	{"TIFF-BE", []byte{0x4D, 0x4D, 0x00, 0x2A}},      // TIFF big-endian
	{"SQLite", []byte("SQLite format 3\x00")},        // SQLite database
	{"7Z", []byte{'7', 'z', 0xBC, 0xAF, 0x27, 0x1C}}, // 7-Zip archive
	{"RAR", []byte("Rar!\x1A\x07")},                  // RAR archive
	{"class", []byte{0xCA, 0xFE, 0xBA, 0xBE}},        // Java class
	{"Mach-O 32", []byte{0xFE, 0xED, 0xFA, 0xCE}},    // macOS executable
	{"Mach-O 64", []byte{0xFE, 0xED, 0xFA, 0xCF}},    // macOS 64-bit executable
}

// nullScanLimit caps the number of bytes scanned for null bytes.
// 8 KB is enough to reliably detect binary content without scanning
// the entire file, keeping the check O(1) for large inputs.
const nullScanLimit = 8 * 1024

// ValidateContent checks that data is valid text suitable for CalcMark.
// It rejects known binary formats (magic-number check), null bytes, and
// data that is not valid UTF-8. This is a defense-in-depth measure: even
// if a file has a .cm extension, its content must be safe to interpret.
func ValidateContent(data []byte) error {
	// 1. Magic-number check: reject known binary file formats.
	for _, sig := range knownBinarySignatures {
		if bytes.HasPrefix(data, sig.magic) {
			return fmt.Errorf("file is not valid CalcMark: detected %s binary content", sig.name)
		}
	}

	// 2. Null-byte check: text files must not contain 0x00.
	scanLen := min(len(data), nullScanLimit)
	if bytes.ContainsRune(data[:scanLen], '\x00') {
		return fmt.Errorf("file is not valid CalcMark: contains null bytes (binary content)")
	}

	// 3. UTF-8 validation: CalcMark is a text format; all content must be
	// valid UTF-8.
	if !utf8.Valid(data) {
		return fmt.Errorf("file is not valid CalcMark: content is not valid UTF-8 text")
	}

	return nil
}
