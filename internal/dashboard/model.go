package dashboard

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/autor-dev/dev-worktree/internal/container"
)

// TranscriptLine represents a formatted line from Claude's session log.
type TranscriptLine struct {
	Type string // "user", "asst", "tool", "done"
	Text string
}

// Model is the main bubbletea model for the dashboard.
type Model struct {
	dc         *container.Client
	envs       []container.Environment
	selected   int
	transcript []TranscriptLine
	width      int
	height     int
	err        error
	quitting   bool
	// Track which container we're streaming transcript for, so we restart
	// streaming when the selection changes.
	streamingContainerID string
}

// Message types.

type envsUpdatedMsg struct {
	envs []container.Environment
}

type transcriptMsg struct {
	containerID string
	lines       []TranscriptLine
}

type errMsg struct {
	err error
}

// New creates a new dashboard model.
func New(dc *container.Client) Model {
	return Model{
		dc: dc,
	}
}

// Init starts the initial environment refresh.
func (m Model) Init() tea.Cmd {
	return refreshEnvs(m.dc)
}

// Update handles messages and key events.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "up", "k":
			if m.selected > 0 {
				m.selected--
				m.transcript = nil
				return m, m.startTranscriptStream()
			}
		case "down", "j":
			if m.selected < len(m.envs)-1 {
				m.selected++
				m.transcript = nil
				return m, m.startTranscriptStream()
			}
		case "o":
			m.openInBrowser()
		}
		return m, nil

	case envsUpdatedMsg:
		m.envs = msg.envs
		if m.selected >= len(m.envs) && len(m.envs) > 0 {
			m.selected = len(m.envs) - 1
		}
		var cmds []tea.Cmd
		// Schedule next env refresh in 3 seconds.
		cmds = append(cmds, tea.Tick(3*time.Second, func(time.Time) tea.Msg {
			return tickEnvsMsg{}
		}))
		// Start transcript stream if not already streaming the selected container.
		if cmd := m.startTranscriptStream(); cmd != nil {
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	case tickEnvsMsg:
		return m, refreshEnvs(m.dc)

	case transcriptMsg:
		// Only accept transcript for the currently selected container.
		if m.selectedContainerID() == msg.containerID {
			m.transcript = msg.lines
			// Schedule next transcript poll in 2 seconds.
			containerID := msg.containerID
			return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
				return tickTranscriptMsg{containerID: containerID}
			})
		}
		return m, nil

	case tickTranscriptMsg:
		if m.selectedContainerID() == msg.containerID {
			return m, streamTranscript(m.dc, msg.containerID)
		}
		return m, nil

	case errMsg:
		m.err = msg.err
		return m, nil
	}

	return m, nil
}

// View renders the full dashboard.
func (m Model) View() string {
	if m.quitting {
		return ""
	}
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	left := m.renderStatus()
	right := m.renderTranscript()

	return joinHorizontal(left, right, m.width, m.height)
}

// Helper types for tick messages.

type tickEnvsMsg struct{}

type tickTranscriptMsg struct {
	containerID string
}

// selectedContainerID returns the container ID of the currently selected environment.
func (m Model) selectedContainerID() string {
	if m.selected >= 0 && m.selected < len(m.envs) {
		return m.envs[m.selected].Container
	}
	return ""
}

// startTranscriptStream starts streaming if the selected container changed.
func (m *Model) startTranscriptStream() tea.Cmd {
	cid := m.selectedContainerID()
	if cid == "" || cid == m.streamingContainerID {
		return nil
	}
	m.streamingContainerID = cid
	return streamTranscript(m.dc, cid)
}

// openInBrowser opens the first port of the selected environment in the default browser.
func (m Model) openInBrowser() {
	if m.selected < 0 || m.selected >= len(m.envs) {
		return
	}
	env := m.envs[m.selected]
	if len(env.Ports) == 0 {
		return
	}
	url := fmt.Sprintf("http://localhost:%d", env.Ports[0].HostPort)
	_ = exec.Command("open", url).Start()
}

// refreshEnvs calls dc.List() and returns the result as a message.
func refreshEnvs(dc *container.Client) tea.Cmd {
	return func() tea.Msg {
		envs, err := dc.List(context.Background())
		if err != nil {
			return errMsg{err: err}
		}
		return envsUpdatedMsg{envs: envs}
	}
}
