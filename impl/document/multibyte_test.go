package document

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/document"
)

// TestMultibyteDocumentEvaluation validates that a CalcMark document containing
// multi-byte UTF-8 content (CJK, Thai, emoji) parses and evaluates correctly.
// This directly validates the sample document from issue #12.
func TestMultibyteDocumentEvaluation(t *testing.T) {
	source := `# 多字节文档测试

This is plain ASCII text.

a = 3
手 = a * 5

就明細今士亮上封訴蝸花果但入東

給料 = 5000
收入 = 給料 * 12

Lorem Ipsum คือ เนื้อหาจำลองแบบเรียบๆ

café = 100
total = 手 + café`

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}

	eval := NewEvaluator()
	err = eval.Evaluate(doc)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	// Verify variables are set correctly in the environment
	env := eval.GetEnvironment()

	expectations := map[string]string{
		"a":  "3",
		"手":  "15",
		"給料": "5000",
		"收入": "60000",
		"café": "100",
	}

	for varName, expected := range expectations {
		val, ok := env.Get(varName)
		if !ok {
			t.Errorf("Variable %q not found in environment", varName)
			continue
		}
		actual := val.String()
		if actual != expected {
			t.Errorf("Variable %q: got %q, want %q", varName, actual, expected)
		} else {
			t.Logf("Variable %q = %s", varName, actual)
		}
	}

	// Verify total = 手(15) + café(100) = 115
	totalVal, ok := env.Get("total")
	if !ok {
		t.Error("Variable 'total' not found in environment")
	} else {
		actual := totalVal.String()
		if actual != "115" {
			t.Errorf("Variable 'total': got %q, want %q", actual, "115")
		} else {
			t.Logf("Variable 'total' = %s (cross-script reference works)", actual)
		}
	}
}

// TestMultibyteDocumentBlockClassification validates that blocks containing
// multi-byte prose are correctly classified as text blocks, and calculation
// lines with multi-byte identifiers are classified as calc blocks.
func TestMultibyteDocumentBlockClassification(t *testing.T) {
	source := `# 她鳥足飽經半方結己平向說眼虎

This is plain ascii.

a = 3
手 = a * 5

就明細今士亮上封訴蝸花果但入東

雨久昔邊口，瓜游大兌立呀未南良止寺您木去八氣南追

Lorem Ipsum คือ เนื้อหาจำลองแบบเรียบๆ ที่ใช้กันในธุรกิจงานพิมพ์`

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}

	blocks := doc.GetBlocks()

	// Count calc blocks and text blocks
	calcCount := 0
	textCount := 0
	for _, node := range blocks {
		switch node.Block.(type) {
		case *document.CalcBlock:
			calcCount++
		case *document.TextBlock:
			textCount++
		}
	}

	if calcCount == 0 {
		t.Error("Expected at least one calc block (a=3, 手=a*5)")
	}
	if textCount == 0 {
		t.Error("Expected at least one text block (CJK/Thai prose)")
	}

	t.Logf("Document has %d calc blocks and %d text blocks", calcCount, textCount)
}

// TestMultibyteDocumentVariableScoping validates that multi-byte variables
// assigned in one calc block are accessible in later blocks (global scope).
func TestMultibyteDocumentVariableScoping(t *testing.T) {
	// Two calc blocks separated by a hard boundary (two empty lines)
	source := `手 = 10


result = 手 * 2`

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}

	eval := NewEvaluator()
	err = eval.Evaluate(doc)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	env := eval.GetEnvironment()

	resultVal, ok := env.Get("result")
	if !ok {
		t.Fatal("Variable 'result' not found - CJK variable not accessible across blocks")
	}

	actual := resultVal.String()
	if actual != "20" {
		t.Errorf("result = %q, want %q", actual, "20")
	} else {
		t.Logf("Cross-block CJK variable reference works: result = %s", actual)
	}
}
