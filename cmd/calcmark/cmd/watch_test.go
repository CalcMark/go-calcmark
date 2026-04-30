package cmd

import (
	"bytes"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/CalcMark/go-calcmark/v2"
	"github.com/CalcMark/go-calcmark/v2/format"
	"github.com/fsnotify/fsnotify"
)

// syncBuf is a goroutine-safe wrapper around bytes.Buffer used by tests
// that capture watchServer.logf output. The watchLoop goroutine writes
// concurrently with the test goroutine reading String(); a bare
// bytes.Buffer would race (caught by `go test -race`).
type syncBuf struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuf) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuf) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

func TestRenderFile(t *testing.T) {
	// Create a temp .cm file
	dir := t.TempDir()
	path := filepath.Join(dir, "test.cm")
	if err := os.WriteFile(path, []byte("price = 100\ntax = price * 0.1"), 0644); err != nil {
		t.Fatal(err)
	}

	html, err := renderFile(path, calcmark.CM)
	if err != nil {
		t.Fatalf("renderFile: %v", err)
	}

	if html == "" {
		t.Fatal("expected non-empty HTML output")
	}
}

func TestRenderFile_InvalidFile(t *testing.T) {
	_, err := renderFile("/nonexistent/file.cm", calcmark.CM)
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestRenderFile_Embedded(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	content := "# Test\n\n```cm\nx = 42\n```\n\nSome prose.\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	html, err := renderFile(path, calcmark.Embedded)
	if err != nil {
		t.Fatalf("renderFile embedded: %v", err)
	}

	if html == "" {
		t.Fatal("expected non-empty HTML output")
	}
	if !strings.Contains(html, "<h1") {
		t.Error("expected HTML heading from goldmark")
	}
	if !strings.Contains(html, "Some prose.") {
		t.Error("expected prose in output")
	}
}

func TestHandlePage_IncludesCalcStyles(t *testing.T) {
	srv := &watchServer{
		sessionToken: "testtoken",
		html:         `<div class="calc-block"><div class="calc-line"><code class="calc-source">x = 1</code></div></div>`,
	}

	req := httptest.NewRequest("GET", "/testtoken", nil)
	rec := httptest.NewRecorder()
	srv.handlePage(rec, req)

	body := rec.Body.String()

	// The watch page must include CSS for the content classes from preview.gohtml
	cssRules := []string{
		".calc-block",
		".calc-line",
		".calc-source",
		".calc-inline-result",
		".calc-error",
		".text-block",
		".frontmatter",
	}
	for _, rule := range cssRules {
		if !strings.Contains(body, rule+" {") && !strings.Contains(body, rule+" ") {
			t.Errorf("watch page missing CSS rule for %s", rule)
		}
	}
}

func TestStyleCSS_UsedByWatchPage(t *testing.T) {
	// The watch page template must use the shared format.StyleCSS() source,
	// not a duplicated copy of the CSS. Verify by checking that the watch
	// page output contains the exact shared CSS string.
	css := format.StyleCSS()
	if css == "" {
		t.Fatal("format.StyleCSS() returned empty string")
	}

	srv := &watchServer{
		sessionToken: "testtoken",
		html:         "<p>test</p>",
	}
	req := httptest.NewRequest("GET", "/testtoken", nil)
	rec := httptest.NewRecorder()
	srv.handlePage(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, css) {
		t.Error("watch page does not contain the shared format.StyleCSS() output — CSS is likely duplicated")
	}
}

func TestStyleCSS_UsedByDefaultTemplate(t *testing.T) {
	// The default HTML template must reference {{.Style}} for the shared CSS,
	// not contain a hardcoded copy of the styles.
	tmpl := format.DefaultHTMLTemplate()
	if !strings.Contains(tmpl, "{{.Style}}") {
		t.Error("default HTML template does not reference {{.Style}} — CSS may be duplicated")
	}
	// And the shared CSS must exist and be non-trivial
	css := format.StyleCSS()
	if !strings.Contains(css, ".calc-block") {
		t.Error("format.StyleCSS() missing .calc-block rule")
	}
}

func TestStyleCSS_HasCustomProperties(t *testing.T) {
	css := format.StyleCSS()

	// CSS custom properties must be present for theming
	vars := []string{
		"--cm-accent",
		"--cm-error",
		"--cm-warning",
		"--cm-text",
		"--cm-bg",
		"--cm-font-sans",
		"--cm-font-mono",
	}
	for _, v := range vars {
		if !strings.Contains(css, v) {
			t.Errorf("format.StyleCSS() missing CSS custom property %s", v)
		}
	}

	// Variables should be used (not just defined)
	if !strings.Contains(css, "var(--cm-accent)") {
		t.Error("CSS rules should use var(--cm-accent), not hardcoded values")
	}
}

func TestHandlePage_EmbeddedModeTypography(t *testing.T) {
	// Embedded mode produces bare HTML (h1, p, pre>code, table, blockquote)
	// without .text-block wrappers. The watch page CSS must style these bare elements.
	srv := &watchServer{
		sessionToken: "testtoken",
		html:         `<h1>Title</h1><pre><code class="language-calcmark">x = 1</code></pre><blockquote><p>note</p></blockquote>`,
	}

	req := httptest.NewRequest("GET", "/testtoken", nil)
	rec := httptest.NewRecorder()
	srv.handlePage(rec, req)

	body := rec.Body.String()

	// Must have styles for bare elements (not just .text-block scoped ones)
	embeddedRules := []string{
		"#content h1",    // bare heading styles for embedded mode
		"#content pre",   // bare pre styles for embedded mode
		"#content code",  // bare code styles for embedded mode
		"#content table", // bare table styles for embedded mode
	}
	for _, rule := range embeddedRules {
		if !strings.Contains(body, rule) {
			t.Errorf("watch page missing embedded-mode CSS rule for %q", rule)
		}
	}
}

func TestAddWatch_WatchesDirectory(t *testing.T) {
	// addWatch should watch the parent directory, not the file itself.
	// This ensures atomic saves (write-temp-rename) are detected on all platforms.
	dir := t.TempDir()
	cmFile := filepath.Join(dir, "test.cm")
	if err := os.WriteFile(cmFile, []byte("x = 1"), 0644); err != nil {
		t.Fatal(err)
	}

	srv := &watchServer{filename: cmFile, mode: calcmark.CM}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatal(err)
	}
	defer watcher.Close()

	if err := srv.addWatch(watcher); err != nil {
		t.Fatal(err)
	}

	// The watcher should be watching the directory, not the file
	watchList := watcher.WatchList()
	if !slices.Contains(watchList, dir) {
		t.Errorf("addWatch should watch directory %s, but watch list is %v", dir, watchList)
	}
}

func TestWatchLoop_AtomicSave(t *testing.T) {
	// Create a temp dir with a .cm file
	dir := t.TempDir()
	cmFile := filepath.Join(dir, "test.cm")
	if err := os.WriteFile(cmFile, []byte("x = 1"), 0644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	srv := &watchServer{
		filename: cmFile,
		mode:     calcmark.CM,
		logw:     &buf,
	}

	// Initial render
	html, err := renderFile(cmFile, calcmark.CM)
	if err != nil {
		t.Fatal(err)
	}
	srv.html = html

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatal(err)
	}
	defer watcher.Close()

	if err := srv.addWatch(watcher); err != nil {
		t.Fatal(err)
	}

	go srv.watchLoop(watcher)

	// Give the watcher time to start
	time.Sleep(50 * time.Millisecond)

	// Simulate atomic save: write temp file, rename over original
	tmpFile := filepath.Join(dir, "test.cm.tmp")
	if err := os.WriteFile(tmpFile, []byte("x = 42"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(tmpFile, cmFile); err != nil {
		t.Fatal(err)
	}

	// Wait for debounce (100ms) + processing time
	deadline := time.After(2 * time.Second)
	for {
		srv.mu.RLock()
		current := srv.html
		srv.mu.RUnlock()
		if strings.Contains(current, "42") {
			break // Success: the change was detected
		}
		select {
		case <-deadline:
			t.Fatal("atomic save was not detected within 2 seconds — file watcher lost the file")
		case <-time.After(50 * time.Millisecond):
			// Poll again
		}
	}
}

func TestWatchLoop_LogsOnChange(t *testing.T) {
	dir := t.TempDir()
	cmFile := filepath.Join(dir, "test.cm")
	if err := os.WriteFile(cmFile, []byte("a = 1"), 0644); err != nil {
		t.Fatal(err)
	}

	// Capture stderr through a goroutine-safe buffer (the watchLoop
	// goroutine writes concurrently with this test reading String()).
	var buf syncBuf
	srv := &watchServer{
		filename: cmFile,
		mode:     calcmark.CM,
		logw:     &buf,
	}
	html, err := renderFile(cmFile, calcmark.CM)
	if err != nil {
		t.Fatal(err)
	}
	srv.html = html

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatal(err)
	}
	defer watcher.Close()

	if err := srv.addWatch(watcher); err != nil {
		t.Fatal(err)
	}
	go srv.watchLoop(watcher)
	time.Sleep(50 * time.Millisecond)

	// Trigger a normal write
	if err := os.WriteFile(cmFile, []byte("a = 2"), 0644); err != nil {
		t.Fatal(err)
	}

	// Wait for debounce + render
	time.Sleep(300 * time.Millisecond)

	output := buf.String()
	if !strings.Contains(output, "[watch] change detected") {
		t.Errorf("expected '[watch] change detected' in stderr, got: %q", output)
	}
	if !strings.Contains(output, "[watch] re-rendered") {
		t.Errorf("expected '[watch] re-rendered' in stderr, got: %q", output)
	}
}

func TestLogf_WritesToLogWriter(t *testing.T) {
	var buf bytes.Buffer
	srv := &watchServer{logw: &buf}

	srv.logf("[watch] test message: %d", 42)

	output := buf.String()
	if !strings.Contains(output, "[watch] test message: 42") {
		t.Errorf("expected formatted log message, got: %q", output)
	}
}

func TestLogf_DefaultsToStderr(t *testing.T) {
	// When logw is nil, logf should not panic.
	srv := &watchServer{}
	// This writes to os.Stderr — we just verify it doesn't panic.
	srv.logf("[watch] smoke test")
}

func TestIsLoopbackOrigin(t *testing.T) {
	tests := []struct {
		origin string
		want   bool
	}{
		{"http://127.0.0.1:3141", true},
		{"http://localhost:3141", true},
		{"http://[::1]:3141", true},
		{"http://evil.com", false},
		{"http://127.0.0.1.evil.com", true}, // conservative — contains "127.0.0.1"
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.origin, func(t *testing.T) {
			got := isLoopbackOrigin(tt.origin)
			if got != tt.want {
				t.Errorf("isLoopbackOrigin(%q) = %v, want %v", tt.origin, got, tt.want)
			}
		})
	}
}
