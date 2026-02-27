package filecheck

import (
	"strings"
	"testing"
)

// --- IsCalcMarkExtension tests ---

func TestIsCalcMarkExtension(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"budget.cm", true},
		{"budget.calcmark", true},
		{"BUDGET.CM", true},
		{"BUDGET.CALCMARK", true},
		{"budget.Cm", true},
		{"/path/to/budget.cm", true},
		{"budget.md", false},
		{"budget.txt", false},
		{"budget.json", false},
		{"budget", false},        // no extension
		{"budget.cm.bak", false}, // double extension
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := IsCalcMarkExtension(tt.path); got != tt.want {
				t.Errorf("IsCalcMarkExtension(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

// --- Magic number rejection tests ---

func TestValidateContent_RejectsPNG(t *testing.T) {
	// Minimal PNG header
	data := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	err := ValidateContent(data)
	if err == nil {
		t.Fatal("expected error for PNG data, got nil")
	}
	if !strings.Contains(err.Error(), "PNG") {
		t.Fatalf("expected PNG in error, got: %v", err)
	}
}

func TestValidateContent_RejectsJPEG(t *testing.T) {
	data := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10}
	err := ValidateContent(data)
	if err == nil {
		t.Fatal("expected error for JPEG data, got nil")
	}
	if !strings.Contains(err.Error(), "JPEG") {
		t.Fatalf("expected JPEG in error, got: %v", err)
	}
}

func TestValidateContent_RejectsGIF(t *testing.T) {
	data := []byte("GIF89a\x01\x00\x01\x00\x80\x00\x00")
	err := ValidateContent(data)
	if err == nil {
		t.Fatal("expected error for GIF data, got nil")
	}
	if !strings.Contains(err.Error(), "GIF") {
		t.Fatalf("expected GIF in error, got: %v", err)
	}
}

func TestValidateContent_RejectsPDF(t *testing.T) {
	data := []byte("%PDF-1.4 fake pdf content")
	err := ValidateContent(data)
	if err == nil {
		t.Fatal("expected error for PDF data, got nil")
	}
	if !strings.Contains(err.Error(), "PDF") {
		t.Fatalf("expected PDF in error, got: %v", err)
	}
}

func TestValidateContent_RejectsZIP(t *testing.T) {
	data := []byte("PK\x03\x04some zip data")
	err := ValidateContent(data)
	if err == nil {
		t.Fatal("expected error for ZIP data, got nil")
	}
	if !strings.Contains(err.Error(), "ZIP") {
		t.Fatalf("expected ZIP in error, got: %v", err)
	}
}

func TestValidateContent_RejectsGZIP(t *testing.T) {
	data := []byte{0x1F, 0x8B, 0x08, 0x00}
	err := ValidateContent(data)
	if err == nil {
		t.Fatal("expected error for GZIP data, got nil")
	}
	if !strings.Contains(err.Error(), "GZIP") {
		t.Fatalf("expected GZIP in error, got: %v", err)
	}
}

func TestValidateContent_RejectsELF(t *testing.T) {
	data := []byte{0x7F, 0x45, 0x4C, 0x46, 0x02, 0x01, 0x01, 0x00}
	err := ValidateContent(data)
	if err == nil {
		t.Fatal("expected error for ELF data, got nil")
	}
	if !strings.Contains(err.Error(), "ELF") {
		t.Fatalf("expected ELF in error, got: %v", err)
	}
}

func TestValidateContent_RejectsPE(t *testing.T) {
	data := []byte("MZ\x90\x00\x03\x00\x00\x00")
	err := ValidateContent(data)
	if err == nil {
		t.Fatal("expected error for PE/MZ data, got nil")
	}
	if !strings.Contains(err.Error(), "PE/MZ") {
		t.Fatalf("expected PE/MZ in error, got: %v", err)
	}
}

func TestValidateContent_RejectsWASM(t *testing.T) {
	data := []byte{0x00, 0x61, 0x73, 0x6D, 0x01, 0x00, 0x00, 0x00}
	err := ValidateContent(data)
	if err == nil {
		t.Fatal("expected error for WASM data, got nil")
	}
	if !strings.Contains(err.Error(), "WASM") {
		t.Fatalf("expected WASM in error, got: %v", err)
	}
}

func TestValidateContent_RejectsRIFF(t *testing.T) {
	data := []byte("RIFF\x24\x00\x00\x00WAVEfmt ")
	err := ValidateContent(data)
	if err == nil {
		t.Fatal("expected error for RIFF data, got nil")
	}
	if !strings.Contains(err.Error(), "RIFF") {
		t.Fatalf("expected RIFF in error, got: %v", err)
	}
}

func TestValidateContent_RejectsSQLite(t *testing.T) {
	data := []byte("SQLite format 3\x00")
	err := ValidateContent(data)
	if err == nil {
		t.Fatal("expected error for SQLite data, got nil")
	}
	if !strings.Contains(err.Error(), "SQLite") {
		t.Fatalf("expected SQLite in error, got: %v", err)
	}
}

func TestValidateContent_RejectsJavaClass(t *testing.T) {
	data := []byte{0xCA, 0xFE, 0xBA, 0xBE, 0x00, 0x00, 0x00, 0x34}
	err := ValidateContent(data)
	if err == nil {
		t.Fatal("expected error for Java class data, got nil")
	}
	if !strings.Contains(err.Error(), "class") {
		t.Fatalf("expected class in error, got: %v", err)
	}
}

func TestValidateContent_RejectsMachO(t *testing.T) {
	// Mach-O 64-bit
	data := []byte{0xFE, 0xED, 0xFA, 0xCF, 0x00, 0x00, 0x00, 0x00}
	err := ValidateContent(data)
	if err == nil {
		t.Fatal("expected error for Mach-O data, got nil")
	}
	if !strings.Contains(err.Error(), "Mach-O") {
		t.Fatalf("expected Mach-O in error, got: %v", err)
	}
}

// --- Null byte rejection tests ---

func TestValidateContent_RejectsNullBytes(t *testing.T) {
	data := []byte("hello\x00world")
	err := ValidateContent(data)
	if err == nil {
		t.Fatal("expected error for null bytes, got nil")
	}
	if !strings.Contains(err.Error(), "null bytes") {
		t.Fatalf("expected null bytes error, got: %v", err)
	}
}

func TestValidateContent_RejectsNullBytesAtStart(t *testing.T) {
	data := []byte("\x00 = some value")
	err := ValidateContent(data)
	if err == nil {
		t.Fatal("expected error for leading null byte, got nil")
	}
}

func TestValidateContent_RejectsNullBytesWithinScanLimit(t *testing.T) {
	// Null byte at position 4096 (within 8KB scan limit)
	data := make([]byte, 8*1024)
	for i := range data {
		data[i] = 'x'
	}
	data[4096] = 0x00
	err := ValidateContent(data)
	if err == nil {
		t.Fatal("expected error for null byte within scan limit, got nil")
	}
}

func TestValidateContent_NullBytesBeyondScanLimit(t *testing.T) {
	// Null byte well beyond the 8KB scan limit — still caught by UTF-8
	// check if the byte appears in an invalid UTF-8 sequence, but a lone
	// 0x00 is valid UTF-8 (U+0000). This tests the scan-limit boundary.
	data := make([]byte, 16*1024)
	for i := range data {
		data[i] = 'x'
	}
	data[9000] = 0x00
	// A lone null byte is technically valid UTF-8, so only the scan-window
	// catches it. Bytes past 8KB won't be scanned for nulls.
	err := ValidateContent(data)
	if err != nil {
		t.Fatalf("null byte beyond scan limit should not be caught by null scan, got: %v", err)
	}
}

// --- UTF-8 validation tests ---

func TestValidateContent_RejectsInvalidUTF8(t *testing.T) {
	// Invalid UTF-8: 0xFE is never valid in UTF-8
	data := []byte{0xFE, 0xFE, 0xFF, 0xFF}
	err := ValidateContent(data)
	if err == nil {
		t.Fatal("expected error for invalid UTF-8, got nil")
	}
	if !strings.Contains(err.Error(), "UTF-8") {
		t.Fatalf("expected UTF-8 error, got: %v", err)
	}
}

func TestValidateContent_RejectsInvalidUTF8Continuation(t *testing.T) {
	// Start of a 2-byte UTF-8 sequence but missing continuation byte
	data := []byte("valid text\xC0")
	err := ValidateContent(data)
	if err == nil {
		t.Fatal("expected error for truncated UTF-8 sequence, got nil")
	}
	if !strings.Contains(err.Error(), "UTF-8") {
		t.Fatalf("expected UTF-8 error, got: %v", err)
	}
}

func TestValidateContent_RejectsLatin1(t *testing.T) {
	// ISO-8859-1 encoded text (0xE9 = 'é' in Latin-1, invalid alone in UTF-8)
	data := []byte("caf\xe9")
	err := ValidateContent(data)
	if err == nil {
		t.Fatal("expected error for Latin-1 encoded text, got nil")
	}
	if !strings.Contains(err.Error(), "UTF-8") {
		t.Fatalf("expected UTF-8 error, got: %v", err)
	}
}

// --- Valid content tests ---

func TestValidateContent_AcceptsEmptyInput(t *testing.T) {
	if err := ValidateContent([]byte{}); err != nil {
		t.Fatalf("expected no error for empty input, got: %v", err)
	}
}

func TestValidateContent_AcceptsSimpleCalcMark(t *testing.T) {
	data := []byte("x = 10\ny = x + 5\n")
	if err := ValidateContent(data); err != nil {
		t.Fatalf("expected no error for valid CalcMark, got: %v", err)
	}
}

func TestValidateContent_AcceptsUTF8Text(t *testing.T) {
	data := []byte("prix = 10\n// café: résultat\ntotal = prix * 2\n")
	if err := ValidateContent(data); err != nil {
		t.Fatalf("expected no error for UTF-8 text, got: %v", err)
	}
}

func TestValidateContent_AcceptsMarkdownWithCalcMark(t *testing.T) {
	data := []byte("# Budget\n\nrent = 1200\nutilities = 150\ntotal = rent + utilities\n")
	if err := ValidateContent(data); err != nil {
		t.Fatalf("expected no error for markdown + calcmark, got: %v", err)
	}
}

func TestValidateContent_AcceptsBOM(t *testing.T) {
	// UTF-8 BOM is valid UTF-8
	data := []byte("\xEF\xBB\xBFx = 1\n")
	if err := ValidateContent(data); err != nil {
		t.Fatalf("expected no error for UTF-8 BOM text, got: %v", err)
	}
}

func TestValidateContent_AcceptsCJK(t *testing.T) {
	// Chinese/Japanese/Korean characters are valid UTF-8
	data := []byte("// 計算\n価格 = 100\n")
	if err := ValidateContent(data); err != nil {
		t.Fatalf("expected no error for CJK text, got: %v", err)
	}
}

func TestValidateContent_AcceptsEmoji(t *testing.T) {
	data := []byte("// 📊 Budget\ntotal = 42\n")
	if err := ValidateContent(data); err != nil {
		t.Fatalf("expected no error for emoji in text, got: %v", err)
	}
}

// --- Edge cases ---

func TestValidateContent_RejectsBinaryGarbage(t *testing.T) {
	// Random binary bytes that are not valid UTF-8
	data := []byte{0x80, 0x81, 0x82, 0x83, 0x84, 0x85}
	err := ValidateContent(data)
	if err == nil {
		t.Fatal("expected error for random binary bytes, got nil")
	}
}

func TestValidateContent_AcceptsTextStartingWithM(t *testing.T) {
	// "M" is the first byte of a PE/MZ signature, but "M" alone (or "My")
	// should not be rejected. Only "MZ" prefix triggers rejection.
	data := []byte("My budget = 100\n")
	if err := ValidateContent(data); err != nil {
		t.Fatalf("expected no error for text starting with 'M', got: %v", err)
	}
}

func TestValidateContent_AcceptsTextStartingWithB(t *testing.T) {
	// "B" is the first byte of BMP signature "BM", but "Budget" should pass.
	data := []byte("Budget = 500\n")
	if err := ValidateContent(data); err != nil {
		t.Fatalf("expected no error for text starting with 'B', got: %v", err)
	}
}

func TestValidateContent_AcceptsTextStartingWithPercent(t *testing.T) {
	// "%" followed by non-"PDF" text should pass.
	data := []byte("% this is a comment\nx = 1\n")
	if err := ValidateContent(data); err != nil {
		t.Fatalf("expected no error for text starting with '%%', got: %v", err)
	}
}

// --- Table-driven test for all magic signatures ---

func TestValidateContent_RejectsAllKnownSignatures(t *testing.T) {
	for _, sig := range knownBinarySignatures {
		t.Run(sig.name, func(t *testing.T) {
			// Pad with valid UTF-8 text so only the magic-number check triggers
			data := make([]byte, len(sig.magic)+20)
			copy(data, sig.magic)
			for i := len(sig.magic); i < len(data); i++ {
				data[i] = 'A'
			}
			err := ValidateContent(data)
			if err == nil {
				t.Fatalf("expected error for %s signature, got nil", sig.name)
			}
			if !strings.Contains(err.Error(), sig.name) {
				t.Fatalf("expected %q in error, got: %v", sig.name, err)
			}
		})
	}
}

// --- Fuzz test ---

func FuzzValidateContent(f *testing.F) {
	// Seed with valid CalcMark
	f.Add([]byte("x = 1\n"))
	f.Add([]byte("# Title\nprice = 100\ntax = price * 0.1\n"))
	// Seed with known binary headers
	f.Add([]byte{0x89, 0x50, 0x4E, 0x47})
	f.Add([]byte{0xFF, 0xD8, 0xFF})
	f.Add([]byte("GIF89a"))
	f.Add([]byte("%PDF-1.7"))
	f.Add([]byte{0x00, 0x61, 0x73, 0x6D})
	// Seed with null bytes
	f.Add([]byte{0x00})
	f.Add([]byte("abc\x00def"))
	// Seed with invalid UTF-8
	f.Add([]byte{0xFE, 0xFF})
	f.Add([]byte{0xC0, 0xAF})
	// Seed with empty
	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, data []byte) {
		// ValidateContent must not panic on any input.
		_ = ValidateContent(data)
	})
}
