//go:build !js && !wasm

package editor

import (
	"errors"
	"fmt"
	"os/exec"

	tea "charm.land/bubbletea/v2"
	"github.com/CalcMark/go-calcmark/v2/cmd/calcmark/tui/editor/store"
	"github.com/atotto/clipboard"
)

func init() {
	// Add store commands to the command menu (native builds only).
	EditorCommands = append(EditorCommands,
		Command{Name: "Share To Gist", Accelerator: "", Description: "Share document as GitHub Gist", Category: "file"},
		Command{Name: "Open From Gist", Accelerator: "", Description: "Open a GitHub Gist", Category: "file"},
	)
}

// executeShareToGist runs the share-to-gist workflow:
// 1. Check gh availability
// 2. Check auth (offer interactive login if needed)
// 3. Share the document and capture the URL
// 4. Copy URL to clipboard
func (m Model) executeShareToGist() (tea.Model, tea.Cmd) {
	gist := store.NewGistStore(store.RealExecutor{})

	// Check CLI availability
	if err := gist.CheckAvailable(); err != nil {
		if errors.Is(err, store.ErrCLINotFound) {
			m.statusMsg = "gh CLI not found. Install: https://cli.github.com"
		} else {
			m.statusMsg = fmt.Sprintf("Store error: %v", err)
		}
		m.statusIsErr = true
		m.exitOverlay()
		return m, nil
	}

	// Check auth — if not authenticated, launch interactive login
	if err := gist.CheckAuth(); err != nil {
		if errors.Is(err, store.ErrNotAuthenticated) {
			m.statusMsg = "Authenticating with GitHub..."
			// Use tea.Exec to give gh terminal control for interactive login
			ghPath, _ := exec.LookPath("gh")
			if ghPath == "" {
				ghPath = "gh"
			}
			c := exec.Command(ghPath, "auth", "login")
			return m, tea.ExecProcess(c, func(err error) tea.Msg {
				if err != nil {
					return shareResultMsg{err: fmt.Errorf("auth failed: %w", err)}
				}
				return retryShareMsg{}
			})
		}
		m.statusMsg = fmt.Sprintf("Auth error: %v", err)
		m.statusIsErr = true
		m.exitOverlay()
		return m, nil
	}

	// Auth is good — perform the share in a goroutine
	content := m.getDocumentContent()
	filename := resolveFilename(m.filepath)
	description := m.shareDescription
	public := m.shareVisibility == 0

	m.statusMsg = "Sharing to Gist..."
	return m, func() tea.Msg {
		result, err := gist.Share(content, filename, description, public)
		if err != nil {
			return shareResultMsg{err: err}
		}
		// Try to copy URL to clipboard
		copied := clipboard.WriteAll(result.URL) == nil
		return shareResultMsg{url: result.URL, copied: copied}
	}
}

// executeOpenFromGist runs the open-from-gist workflow:
// 1. Check gh availability
// 2. Check auth (offer interactive login if needed)
// 3. Fetch gist content
func (m Model) executeOpenFromGist(identifier string) (tea.Model, tea.Cmd) {
	gist := store.NewGistStore(store.RealExecutor{})

	// Check CLI availability
	if err := gist.CheckAvailable(); err != nil {
		if errors.Is(err, store.ErrCLINotFound) {
			m.statusMsg = "gh CLI not found. Install: https://cli.github.com"
		} else {
			m.statusMsg = fmt.Sprintf("Store error: %v", err)
		}
		m.statusIsErr = true
		m.exitOverlay()
		return m, nil
	}

	// Check auth — if not authenticated, launch interactive login
	if err := gist.CheckAuth(); err != nil {
		if errors.Is(err, store.ErrNotAuthenticated) {
			m.statusMsg = "Authenticating with GitHub..."
			ghPath, _ := exec.LookPath("gh")
			if ghPath == "" {
				ghPath = "gh"
			}
			c := exec.Command(ghPath, "auth", "login")
			return m, tea.ExecProcess(c, func(err error) tea.Msg {
				if err != nil {
					return openFromResultMsg{err: fmt.Errorf("auth failed: %w", err)}
				}
				return retryOpenFromMsg{identifier: identifier}
			})
		}
		m.statusMsg = fmt.Sprintf("Auth error: %v", err)
		m.statusIsErr = true
		m.exitOverlay()
		return m, nil
	}

	// Auth is good — fetch the gist in a goroutine
	m.statusMsg = "Fetching Gist..."
	return m, func() tea.Msg {
		result, err := gist.Open(identifier)
		if err != nil {
			return openFromResultMsg{err: err}
		}
		return openFromResultMsg{content: result.Content, filename: result.Filename}
	}
}
