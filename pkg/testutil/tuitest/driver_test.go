package tuitest

import (
	"fmt"
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	tea "github.com/charmbracelet/bubbletea"
)

type lifecycleModel struct{ updates int }

func (m lifecycleModel) Init() tea.Cmd {
	return func() tea.Msg { return lifecycleMsg{} }
}

func (m lifecycleModel) Update(tea.Msg) (tea.Model, tea.Cmd) {
	m.updates++
	return m, nil
}

func (m lifecycleModel) View() string { return fmt.Sprintf("frame-%d", m.updates) }

type lifecycleMsg struct{}

type sizedModel struct{ width, height int }

func (m sizedModel) Init() tea.Cmd { return nil }
func (m sizedModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		m.width, m.height = size.Width, size.Height
	}
	return m, nil
}
func (m sizedModel) View() string { return fmt.Sprintf("%dx%d", m.width, m.height) }

type quittingModel struct{}

func (quittingModel) Init() tea.Cmd { return nil }
func (quittingModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(tea.KeyMsg); ok {
		return quittingModel{}, tea.Quit
	}
	return quittingModel{}, nil
}
func (quittingModel) View() string { return "running" }

func TestDriverRendersInitialCommandAndInputFrames(t *testing.T) {
	t.Parallel()
	driver := NewDriver(t, lifecycleModel{})
	driver.RequireText("frame-1")
	driver.Send(lifecycleMsg{})
	driver.RequireText("frame-2")
	testutil.Equals(t, driver.History(), []string{"frame-0", "frame-1", "frame-2"})
}

func TestDriverResizeRecordsBoundedFrame(t *testing.T) {
	t.Parallel()
	driver := NewDriver(t, sizedModel{})
	driver.Resize(80, 24)
	driver.RequireText("80x24")
	driver.RequireViewport(80, 24)
	testutil.Equals(t, driver.History(), []string{"0x0", "80x24"})
}

func TestDriverObservesProgramTermination(t *testing.T) {
	t.Parallel()
	driver := NewDriver(t, quittingModel{})
	driver.RequireRunning()
	driver.Press("q")
	driver.RequireQuit()
}
