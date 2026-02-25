package editor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/cockroachdb/datadriven"
)

// observerV2 extracts observable state from a v2 model during testing.
type observerV2 func(out io.Writer, m tea.Model) error

// optionV2 configures the v2 test driver.
type optionV2 func(*driverV2)

// WithObserverV2 registers an observer that can be referenced by name
// in testdata "run observe=(...)" directives.
func WithObserverV2(name string, obs observerV2) optionV2 {
	return func(d *driverV2) {
		d.observers[name] = obs
	}
}

// driverV2 is a Bubble Tea v2-compatible replacement for the catwalk test
// driver. It processes the same testdata format produced by
// github.com/cockroachdb/datadriven and supports the same commands (run, key,
// type, resize, enter, paste).
type driverV2 struct {
	ctx    context.Context
	cancel func()

	m tea.Model

	result bytes.Buffer

	// Queued commands left for processing.
	cmds []tea.Cmd

	// cmdTimeout is how long to wait for a tea.Cmd to return a tea.Msg.
	cmdTimeout time.Duration

	// Queued messages left for processing.
	msgs []tea.Msg

	// Named test observers.
	observers map[string]observerV2

	// Whether Init() has been called yet.
	startDone bool

	// pos is the position in the input data file, used for error messages.
	pos string
}

const defaultCmdTimeoutV2 = 20 * time.Millisecond

// newDriverV2 creates a v2 test driver for the given model.
func newDriverV2(m tea.Model, opts ...optionV2) *driverV2 {
	ctx, cancel := context.WithCancel(context.Background())
	d := &driverV2{
		ctx:        ctx,
		cancel:     cancel,
		m:          m,
		cmdTimeout: defaultCmdTimeoutV2,
		observers: map[string]observerV2{
			"view":  observeViewV2,
			"debug": observeDebugV2,
		},
	}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

// RunModelV2 runs the tests contained in the file pointed to by 'path'
// against the v2 model m. It is a drop-in replacement for catwalk.RunModel
// that works with Bubble Tea v2 types.
func RunModelV2(t *testing.T, path string, m tea.Model, opts ...optionV2) {
	t.Helper()
	d := newDriverV2(m, opts...)
	defer d.close()

	datadriven.RunTest(t, path, func(t *testing.T, td *datadriven.TestData) string {
		t.Helper()
		return d.runOneTest(t, td)
	})
}

func (d *driverV2) close() {
	d.cancel()
}

// ---------------------------------------------------------------------------
// Test directive handling
// ---------------------------------------------------------------------------

func (d *driverV2) runOneTest(t *testing.T, td *datadriven.TestData) string {
	d.pos = td.Pos

	switch td.Cmd {
	case "set", "reset":
		return d.handleSet(t, td)
	case "run":
		return d.handleRun(t, td)
	default:
		t.Fatalf("%s: unrecognized test directive: %s", td.Pos, td.Cmd)
		panic("unreachable")
	}
}

func (d *driverV2) handleSet(t *testing.T, td *datadriven.TestData) string {
	reset := td.Cmd == "reset"
	if len(td.CmdArgs) != 1 ||
		(!reset && len(td.CmdArgs[0].Vals) != 1) ||
		(reset && len(td.CmdArgs[0].Vals) != 0) {
		t.Fatalf("%s: invalid syntax", d.pos)
	}
	key := td.CmdArgs[0].Key
	val := ""
	if !reset {
		val = td.CmdArgs[0].Vals[0]
	}

	switch key {
	case "cmd_timeout":
		if reset {
			val = defaultCmdTimeoutV2.String()
		}
		tm, err := time.ParseDuration(val)
		if err != nil {
			t.Fatalf("%s: invalid timeout value: %v", d.pos, err)
		}
		d.cmdTimeout = tm
		val = d.cmdTimeout.String()
	default:
		t.Fatalf("%s: unknown option %q", d.pos, key)
	}
	if reset {
		return "ok"
	}
	return fmt.Sprintf("%s: %s", key, val)
}

func (d *driverV2) handleRun(t *testing.T, td *datadriven.TestData) string {
	d.result.Reset()

	// Determine which observers to use. Default is just "view".
	var observe []string
	seen := false
	for i := range td.CmdArgs {
		if td.CmdArgs[i].Key == "observe" {
			observe = td.CmdArgs[i].Vals
			seen = true
			break
		}
	}
	if !seen {
		observe = []string{"view"}
	}

	traceEnabled := td.HasArg("trace")
	trace := func(format string, args ...any) {
		if traceEnabled {
			fmt.Fprintf(&d.result, "-- trace: "+format+"\n", args...)
		}
	}

	doObserve := func() {
		for _, obs := range observe {
			o := d.observe(t, obs)
			d.result.WriteString(o)
			if d.result.Len() > 0 && d.result.Bytes()[d.result.Len()-1] != '\n' {
				d.result.WriteByte('\n')
			}
		}
	}

	// Lazy initialisation on first "run" directive.
	if !d.startDone {
		trace("calling Init")
		d.addCmds(d.m.Init())
		d.processTeaCmds(traceEnabled)
		d.startDone = true
	}

	// Process the commands in the test's input.
	for testInputCmd := range strings.SplitSeq(td.Input, "\n") {
		testInputCmd = strings.TrimSpace(testInputCmd)
		if testInputCmd == "" || strings.HasPrefix(testInputCmd, "#") {
			continue
		}

		trace("before %q", testInputCmd)

		// Process any messages produced by the prior command.
		d.processTeaMsgs(traceEnabled)

		// Parse and apply the new text command.
		args := strings.Split(testInputCmd, " ")
		cmd := args[0]
		args = args[1:]
		teaCmd := d.applyTextCommand(t, cmd, args...)
		d.addCmds(teaCmd)
		d.processTeaCmds(traceEnabled)

		if traceEnabled {
			trace("after %q", testInputCmd)
			doObserve()
		}
	}

	if traceEnabled {
		trace("before finish")
		doObserve()
	}

	// Final round of command execution.
	d.processTeaMsgs(traceEnabled)
	d.processTeaCmds(traceEnabled)
	d.processTeaMsgs(traceEnabled)

	trace("at end")
	doObserve()
	return d.result.String()
}

// ---------------------------------------------------------------------------
// Observer support
// ---------------------------------------------------------------------------

func (d *driverV2) observe(t *testing.T, what string) string {
	var buf strings.Builder
	fmt.Fprintf(&buf, "-- %s:\n", what)

	switch what {
	case "msgs":
		fmt.Fprintf(&buf, "msg queue sz: %d\n", len(d.msgs))
		for i, msg := range d.msgs {
			fmt.Fprintf(&buf, "%d:%T: %v\n", i, msg, msg)
		}
	case "cmds":
		fmt.Fprintf(&buf, "command queue sz: %d\n", len(d.cmds))
	default:
		obs, ok := d.observers[what]
		if !ok {
			t.Fatalf("%s: unsupported observer %q, did you call WithObserverV2()?", d.pos, what)
		}
		if err := obs(&buf, d.m); err != nil {
			t.Fatalf("%s: observing %q: %v", d.pos, what, err)
		}
	}
	return buf.String()
}

// observeViewV2 is the built-in "view" observer. It renders the model's view
// with newlines replaced by the visible "␤" marker, matching catwalk v1.
func observeViewV2(buf io.Writer, m tea.Model) error {
	o := m.View().Content
	o = strings.ReplaceAll(o, "\n", "␤\n")
	if len(o) == 0 || o[len(o)-1] != '\n' {
		o += "🛇"
	}
	_, err := io.WriteString(buf, o)
	return err
}

// observeDebugV2 is the built-in "debug" observer. The model must implement
// a Debug() string method.
func observeDebugV2(buf io.Writer, m tea.Model) error {
	type dbg interface{ Debug() string }
	md, ok := m.(dbg)
	if !ok {
		return errors.New("model does not support a Debug() string method")
	}
	_, err := io.WriteString(buf, md.Debug())
	return err
}

// ---------------------------------------------------------------------------
// Command / Message processing
// ---------------------------------------------------------------------------

func (d *driverV2) addCmds(cmds ...tea.Cmd) {
	for _, cmd := range cmds {
		if cmd != nil {
			d.cmds = append(d.cmds, cmd)
		}
	}
}

func (d *driverV2) addMsg(msg tea.Msg) {
	if msg != nil {
		d.msgs = append(d.msgs, msg)
	}
}

func (d *driverV2) processTeaCmds(trace bool) {
	if trace && len(d.cmds) > 0 {
		fmt.Fprintf(&d.result, "-- trace: processing %d cmds\n", len(d.cmds))
	}

	var inputs []tea.Cmd
	for {
		if len(d.cmds) > 0 {
			inputs = append(make([]tea.Cmd, 0, len(d.cmds)+len(inputs)), inputs...)
			inputs = append(inputs, d.cmds...)
			d.cmds = nil
		}
		if len(inputs) == 0 {
			break
		}
		cmd := inputs[0]
		inputs = inputs[1:]
		msg := d.runTeaCmd(cmd, trace)

		if msg == nil {
			continue
		}

		// In v2, BatchMsg is a concrete []tea.Cmd type.
		if batch, ok := msg.(tea.BatchMsg); ok {
			if trace {
				fmt.Fprintf(&d.result, "-- trace: expanded %d commands\n", len(batch))
			}
			d.addCmds(batch...)
			continue
		}

		if trace {
			fmt.Fprintf(&d.result, "-- trace: translated cmd: %T\n", msg)
		}
		d.addMsg(msg)
	}
}

func (d *driverV2) runTeaCmd(cmd tea.Cmd, trace bool) (res tea.Msg) {
	ctx, cancel := context.WithTimeout(d.ctx, d.cmdTimeout)
	defer cancel()

	ch := make(chan tea.Msg, 1)
	go func() {
		ch <- cmd()
	}()
	select {
	case <-ctx.Done():
		if trace {
			fmt.Fprintf(&d.result, "-- trace: timeout waiting for command\n")
		}
	case res = <-ch:
	}
	return res
}

func (d *driverV2) processTeaMsgs(trace bool) {
	if trace && len(d.msgs) > 0 {
		fmt.Fprintf(&d.result, "-- trace: processing %d messages\n", len(d.msgs))
	}
	for _, msg := range d.msgs {
		if trace {
			fmt.Fprintf(&d.result, "-- trace: msg %T\n", msg)
		}

		switch msg.(type) {
		case tea.QuitMsg:
			fmt.Fprintf(&d.result, "TEA QUIT\n")
		case tea.WindowSizeMsg:
			// Window size is visible to the model.
			newM, newCmd := d.m.Update(msg)
			d.m = newM
			d.addCmds(newCmd)
		default:
			newM, newCmd := d.m.Update(msg)
			d.m = newM
			d.addCmds(newCmd)
		}
	}
	d.msgs = d.msgs[:0]
}

// ---------------------------------------------------------------------------
// Text command handling
// ---------------------------------------------------------------------------

func (d *driverV2) applyTextCommand(t *testing.T, cmd string, args ...string) tea.Cmd {
	switch cmd {
	case "resize":
		d.assertArgc(t, args, 2)
		w := d.getInt(t, args[0])
		h := d.getInt(t, args[1])
		d.addMsg(tea.WindowSizeMsg{Width: w, Height: h})

	case "key":
		d.assertArgc(t, args, 1)
		keyName := args[0]
		msg := parseKeyV2(t, d.pos, keyName)
		d.addMsg(msg)

	case "type":
		d.typeIn(args, 0)

	case "enter":
		d.typeIn(args, 0)
		d.addMsg(tea.KeyPressMsg{Code: tea.KeyEnter})

	case "paste":
		arg := strings.Join(args, " ")
		s, err := strconv.Unquote(arg)
		if err != nil {
			t.Fatalf("%s: paste argument error: %v", d.pos, err)
		}
		// In v2, paste is represented by a KeyPressMsg with KeyExtended code
		// and the full text in the Text field (same as typing multiple chars
		// at once via bracketed paste).
		d.addMsg(tea.KeyPressMsg{Code: tea.KeyExtended, Text: s})

	default:
		t.Fatalf("%s: unknown command %q", d.pos, cmd)
	}

	return nil
}

func (d *driverV2) typeIn(args []string, mod tea.KeyMod) {
	var buf strings.Builder
	for i, arg := range args {
		if i > 0 {
			buf.WriteByte(' ')
		}
		buf.WriteString(arg)
	}
	for _, r := range buf.String() {
		d.addMsg(tea.KeyPressMsg{Code: r, Text: string(r), Mod: mod})
	}
}

func (d *driverV2) assertArgc(t *testing.T, args []string, expected int) {
	if len(args) != expected {
		t.Fatalf("%s: expected %d args, got %d", d.pos, expected, len(args))
	}
}

func (d *driverV2) getInt(t *testing.T, v string) int {
	i, err := strconv.Atoi(v)
	if err != nil {
		t.Fatalf("%s: %v", d.pos, err)
	}
	return i
}

// ---------------------------------------------------------------------------
// Key name mapping (v1 catwalk names -> v2 KeyPressMsg)
// ---------------------------------------------------------------------------

// parseKeyV2 translates a catwalk v1 key name (e.g. "ctrl+a", "cmd+left",
// "enter", "up", "space", or a single character like "j") into a Bubble Tea
// v2 KeyPressMsg.
func parseKeyV2(t *testing.T, pos, keyName string) tea.KeyPressMsg {
	var mod tea.KeyMod

	// Strip modifier prefixes.
	remaining := keyName
	for {
		if strings.HasPrefix(remaining, "ctrl+") {
			mod |= tea.ModCtrl
			remaining = strings.TrimPrefix(remaining, "ctrl+")
		} else if strings.HasPrefix(remaining, "alt+") {
			mod |= tea.ModAlt
			remaining = strings.TrimPrefix(remaining, "alt+")
		} else if strings.HasPrefix(remaining, "cmd+") {
			mod |= tea.ModSuper
			remaining = strings.TrimPrefix(remaining, "cmd+")
		} else if strings.HasPrefix(remaining, "shift+") {
			mod |= tea.ModShift
			remaining = strings.TrimPrefix(remaining, "shift+")
		} else {
			break
		}
	}

	// Look up the base key.
	if code, ok := specialKeysV2[remaining]; ok {
		return tea.KeyPressMsg{Code: code, Mod: mod}
	}

	// Single-character keys.
	runes := []rune(remaining)
	if len(runes) == 1 {
		r := runes[0]
		// When a modifier (ctrl/alt/super) is present, the key is a control
		// combo, not printable text.
		if mod != 0 {
			return tea.KeyPressMsg{Code: r, Mod: mod}
		}
		// Plain printable character.
		return tea.KeyPressMsg{Code: r, Text: string(r)}
	}

	t.Fatalf("%s: unknown key: %s", pos, keyName)
	panic("unreachable")
}

// specialKeysV2 maps catwalk v1 key names to v2 key codes. Only special
// (non-character) keys belong here; single-character keys are handled by the
// character path in parseKeyV2.
var specialKeysV2 = map[string]rune{
	// Navigation
	"up":     tea.KeyUp,
	"down":   tea.KeyDown,
	"left":   tea.KeyLeft,
	"right":  tea.KeyRight,
	"home":   tea.KeyHome,
	"end":    tea.KeyEnd,
	"pgup":   tea.KeyPgUp,
	"pgdown": tea.KeyPgDown,

	// Editing
	"enter":     tea.KeyEnter,
	"backspace": tea.KeyBackspace,
	"delete":    tea.KeyDelete,
	"tab":       tea.KeyTab,
	"space":     tea.KeySpace,
	"esc":       tea.KeyEscape,

	// Function keys
	"f1":  tea.KeyF1,
	"f2":  tea.KeyF2,
	"f3":  tea.KeyF3,
	"f4":  tea.KeyF4,
	"f5":  tea.KeyF5,
	"f6":  tea.KeyF6,
	"f7":  tea.KeyF7,
	"f8":  tea.KeyF8,
	"f9":  tea.KeyF9,
	"f10": tea.KeyF10,
	"f11": tea.KeyF11,
	"f12": tea.KeyF12,

	// Insert
	"insert": tea.KeyInsert,
}
