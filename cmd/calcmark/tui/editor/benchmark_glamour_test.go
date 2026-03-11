package editor

import (
	"strings"
	"testing"
)

// Benchmark inputs representing realistic TextBlock content.
var (
	headingLine   = "# System Sizing Calculator"
	paragraphLine = "This calculator helps you estimate the total cost of running a distributed system across multiple availability zones."
	emphasisLine  = "The **total cost** is _significantly_ affected by the `replication factor` you choose."
	listBlock     = "1. Compute instances\n2. Storage volumes\n3. Network egress\n4. Load balancers\n5. DNS queries"
	bulletList    = "- Primary region: us-east-1\n- Secondary region: eu-west-1\n- DR region: ap-southeast-1"
	codeSpanLine  = "Use `read(5 MB, ssd)` to estimate SSD read throughput."
	linkLine      = "See the [AWS pricing page](https://aws.amazon.com/pricing/) for current rates."
	blockquote    = "> Note: All prices are in USD and subject to change."
	tableMD       = "| Region | Instances | Cost |\n|--------|-----------|------|\n| us-east | 10 | $500 |\n| eu-west | 5 | $300 |"
	interpolated  = "The total cost is **$1,234.56** per month across all regions."

	// Mixed block simulating a realistic TextBlock of ~20 lines.
	mixedBlock20 = strings.Join([]string{
		"# Infrastructure Cost Summary",
		"",
		"This document calculates the total infrastructure cost for our",
		"distributed system deployment across three regions.",
		"",
		"## Compute",
		"",
		"- **Primary**: 10 instances at $50/mo each",
		"- **Secondary**: 5 instances at $50/mo each",
		"- **DR**: 2 instances at $50/mo each",
		"",
		"## Storage",
		"",
		"Each instance requires `500 GB` of SSD storage.",
		"The total storage cost is **$850** per month.",
		"",
		"> Note: Prices include reserved instance discounts.",
		"",
		"See the [pricing calculator](https://example.com) for details.",
		"Total monthly cost: **$2,100**",
	}, "\n")

	// ~50 line block for stress testing.
	mixedBlock50 = strings.Repeat(mixedBlock20+"\n\n", 2) + strings.Join([]string{
		"## Network",
		"",
		"| Route | Bandwidth | Cost |",
		"|-------|-----------|------|",
		"| us-east -> eu-west | 100 GB | $9 |",
		"| us-east -> ap-south | 50 GB | $7 |",
		"| eu-west -> ap-south | 25 GB | $4 |",
		"",
		"Total network egress: **$20/mo**",
		"Grand total: **$2,120/mo**",
	}, "\n")
)

// BenchmarkNewMarkdownRenderer measures the cost of creating a new glamour TermRenderer.
func BenchmarkNewMarkdownRenderer(b *testing.B) {
	for b.Loop() {
		r, err := NewMarkdownRenderer(80)
		if err != nil {
			b.Fatal(err)
		}
		_ = r
	}
}

// BenchmarkRenderLine_Heading measures single heading line rendering.
func BenchmarkRenderLine_Heading(b *testing.B) {
	r, _ := NewMarkdownRenderer(80)
	b.ResetTimer()
	for b.Loop() {
		r.RenderLine(headingLine)
	}
}

// BenchmarkRenderLine_Paragraph measures plain paragraph rendering.
func BenchmarkRenderLine_Paragraph(b *testing.B) {
	r, _ := NewMarkdownRenderer(80)
	b.ResetTimer()
	for b.Loop() {
		r.RenderLine(paragraphLine)
	}
}

// BenchmarkRenderLine_Emphasis measures bold/italic/code rendering.
func BenchmarkRenderLine_Emphasis(b *testing.B) {
	r, _ := NewMarkdownRenderer(80)
	b.ResetTimer()
	for b.Loop() {
		r.RenderLine(emphasisLine)
	}
}

// BenchmarkRenderLine_OrderedList measures ordered list rendering.
func BenchmarkRenderLine_OrderedList(b *testing.B) {
	r, _ := NewMarkdownRenderer(80)
	b.ResetTimer()
	for b.Loop() {
		r.RenderLine(listBlock)
	}
}

// BenchmarkRenderLine_BulletList measures unordered list rendering.
func BenchmarkRenderLine_BulletList(b *testing.B) {
	r, _ := NewMarkdownRenderer(80)
	b.ResetTimer()
	for b.Loop() {
		r.RenderLine(bulletList)
	}
}

// BenchmarkRenderLine_Table measures table rendering.
func BenchmarkRenderLine_Table(b *testing.B) {
	r, _ := NewMarkdownRenderer(80)
	b.ResetTimer()
	for b.Loop() {
		r.RenderLine(tableMD)
	}
}

// BenchmarkRenderLine_Blockquote measures blockquote rendering.
func BenchmarkRenderLine_Blockquote(b *testing.B) {
	r, _ := NewMarkdownRenderer(80)
	b.ResetTimer()
	for b.Loop() {
		r.RenderLine(blockquote)
	}
}

// BenchmarkRenderLine_Link measures link rendering.
func BenchmarkRenderLine_Link(b *testing.B) {
	r, _ := NewMarkdownRenderer(80)
	b.ResetTimer()
	for b.Loop() {
		r.RenderLine(linkLine)
	}
}

// BenchmarkRenderLine_Interpolated measures rendering of text with bold values (simulating interpolated {{var}}).
func BenchmarkRenderLine_Interpolated(b *testing.B) {
	r, _ := NewMarkdownRenderer(80)
	b.ResetTimer()
	for b.Loop() {
		r.RenderLine(interpolated)
	}
}

// BenchmarkRenderBlock_20Lines measures rendering a realistic 20-line TextBlock as a single unit.
func BenchmarkRenderBlock_20Lines(b *testing.B) {
	r, _ := NewMarkdownRenderer(80)
	b.ResetTimer()
	for b.Loop() {
		r.RenderLine(mixedBlock20)
	}
}

// BenchmarkRenderBlock_50Lines measures rendering a large 50-line TextBlock.
func BenchmarkRenderBlock_50Lines(b *testing.B) {
	r, _ := NewMarkdownRenderer(80)
	b.ResetTimer()
	for b.Loop() {
		r.RenderLine(mixedBlock50)
	}
}

// BenchmarkRenderBlock_20Lines_WithCreation includes renderer creation cost (worst case: no cache).
func BenchmarkRenderBlock_20Lines_WithCreation(b *testing.B) {
	for b.Loop() {
		r, _ := NewMarkdownRenderer(80)
		r.RenderLine(mixedBlock20)
	}
}

// BenchmarkRenderPerLine_5Lines measures per-line rendering of 5 individual lines.
func BenchmarkRenderPerLine_5Lines(b *testing.B) {
	lines := []string{headingLine, paragraphLine, emphasisLine, codeSpanLine, linkLine}
	r, _ := NewMarkdownRenderer(80)
	b.ResetTimer()
	for b.Loop() {
		for _, line := range lines {
			r.RenderLine(line)
		}
	}
}

// BenchmarkRenderPerLine_20Lines measures per-line rendering of 20 individual lines (current approach).
func BenchmarkRenderPerLine_20Lines(b *testing.B) {
	lines := strings.Split(mixedBlock20, "\n")
	r, _ := NewMarkdownRenderer(80)
	b.ResetTimer()
	for b.Loop() {
		for _, line := range lines {
			r.RenderLine(line)
		}
	}
}
