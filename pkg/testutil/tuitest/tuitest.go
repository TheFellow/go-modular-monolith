package tuitest

import (
	"regexp"
	"strings"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/surfaces/tui/views"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/keyname"
	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// Driver deterministically exercises a complete Bubble Tea model. Unlike the
// focused view helpers below, it renders after every update, including each
// message produced by a command. This mirrors the event-loop boundary where
// intermediate-state rendering failures occur without requiring terminal I/O
// or scheduling goroutines in tests.
type Driver struct {
	t       testing.TB
	model   tea.Model
	screen  string
	history []string
	width   int
	height  int
	quit    bool
}

const commandDrainLimit = 10_000

var sgrWithoutEscape = regexp.MustCompile(`\[(?:[0-9]+;)*[0-9]+m`)

// NewDriver initializes a root model and completely drains its startup work.
func NewDriver(t testing.TB, model tea.Model) *Driver {
	t.Helper()
	d := &Driver{t: t, model: model}
	d.render()
	remaining := commandDrainLimit
	d.drain(model.Init(), &remaining)
	return d
}

// Send routes one message through the root model and drains all resulting work.
func (d *Driver) Send(msg tea.Msg) {
	d.t.Helper()
	if d.quit {
		d.t.Fatal("cannot send input after the Bubble Tea program has quit")
	}
	remaining := commandDrainLimit
	d.send(msg, &remaining)
}

func (d *Driver) send(msg tea.Msg, drained *int) {
	model, cmd := d.model.Update(msg)
	d.model = model
	d.render()
	d.drain(cmd, drained)
}

// Press sends a Bubble Tea key by its conventional name (for example "esc",
// "enter", "ctrl+s", or a string of runes).
func (d *Driver) Press(name string) {
	d.t.Helper()
	var msg tea.KeyMsg
	switch name {
	case keyname.Escape:
		msg = tea.KeyMsg{Type: tea.KeyEsc}
	case keyname.Enter:
		msg = tea.KeyMsg{Type: tea.KeyEnter}
	case keyname.Submit:
		msg = tea.KeyMsg{Type: tea.KeyCtrlS}
	case keyname.Clear:
		msg = tea.KeyMsg{Type: tea.KeyCtrlU}
	case keyname.Up:
		msg = tea.KeyMsg{Type: tea.KeyUp}
	case keyname.Down:
		msg = tea.KeyMsg{Type: tea.KeyDown}
	case keyname.Left:
		msg = tea.KeyMsg{Type: tea.KeyLeft}
	case keyname.Right:
		msg = tea.KeyMsg{Type: tea.KeyRight}
	case keyname.End:
		msg = tea.KeyMsg{Type: tea.KeyEnd}
	default:
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(name)}
	}
	d.Send(msg)
}

// Resize delivers the same message emitted by Bubble Tea for terminal changes.
func (d *Driver) Resize(width, height int) {
	d.t.Helper()
	d.width, d.height = width, height
	remaining := commandDrainLimit
	d.send(tea.WindowSizeMsg{Width: width, Height: height}, &remaining)
}

func (d *Driver) Model() tea.Model { return d.model }
func (d *Driver) Screen() string   { return d.screen }

// RequireRunning asserts that the program has not emitted tea.QuitMsg.
func (d *Driver) RequireRunning() {
	d.t.Helper()
	if d.quit {
		d.t.Fatal("Bubble Tea program unexpectedly quit")
	}
}

// RequireQuit asserts that the program emitted tea.QuitMsg.
func (d *Driver) RequireQuit() {
	d.t.Helper()
	if !d.quit {
		d.t.Fatal("Bubble Tea program is still running")
	}
}

// History returns every rendered frame, including intermediate command frames.
func (d *Driver) History() []string { return append([]string(nil), d.history...) }

func (d *Driver) RequireText(values ...string) {
	d.t.Helper()
	for _, value := range values {
		if !strings.Contains(d.screen, value) {
			d.t.Fatalf("screen does not contain %q:\n%s", value, d.screen)
		}
	}
}

// RequireNoText asserts that none of the values are present in the current frame.
func (d *Driver) RequireNoText(values ...string) {
	d.t.Helper()
	for _, value := range values {
		if strings.Contains(d.screen, value) {
			d.t.Fatalf("screen unexpectedly contains %q:\n%s", value, d.screen)
		}
	}
}

func (d *Driver) RequireViewport(width, height int) {
	d.t.Helper()
	if got := lipgloss.Width(d.screen); got > width {
		d.t.Fatalf("screen width %d exceeds viewport %d:\n%s", got, width, d.screen)
	}
	if got := lipgloss.Height(d.screen); got > height {
		d.t.Fatalf("screen height %d exceeds viewport %d:\n%s", got, height, d.screen)
	}
}

func (d *Driver) render() {
	d.screen = d.model.View()
	d.history = append(d.history, d.screen)
	RequireValidANSI(d.t, d.screen)
	if d.width > 0 && lipgloss.Width(d.screen) > d.width {
		d.t.Fatalf("rendered frame width %d exceeds viewport %d:\n%s", lipgloss.Width(d.screen), d.width, d.screen)
	}
	if d.height > 0 && lipgloss.Height(d.screen) > d.height {
		d.t.Fatalf("rendered frame height %d exceeds viewport %d:\n%s", lipgloss.Height(d.screen), d.height, d.screen)
	}
}

// RequireValidANSI rejects SGR fragments whose escape byte was lost while a
// styled frame was truncated. Valid ANSI sequences are removed first, leaving
// only malformed fragments such as "[38;2;156;163;175m" for detection.
func RequireValidANSI(t testing.TB, rendered string) {
	t.Helper()
	if fragment := malformedANSIFragment(rendered); fragment != "" {
		t.Fatalf("rendered frame contains malformed ANSI fragment %q:\n%s", fragment, rendered)
	}
}

func malformedANSIFragment(rendered string) string {
	return sgrWithoutEscape.FindString(ansi.Strip(rendered))
}

func (d *Driver) drain(cmd tea.Cmd, remaining *int) {
	if cmd == nil {
		return
	}
	if *remaining <= 0 {
		d.t.Fatalf("Bubble Tea command drain exceeded %d messages", commandDrainLimit)
	}
	*remaining--
	msg := cmd()
	if msg == nil {
		return
	}
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, nested := range batch {
			d.drain(nested, remaining)
		}
		return
	}
	if _, ok := msg.(tea.QuitMsg); ok {
		d.quit = true
		return
	}
	// Spinner ticks deliberately schedule themselves forever. One frame is
	// sufficient for deterministic workflow tests; domain-result commands in
	// the same batch are still drained normally.
	if _, ok := msg.(spinner.TickMsg); ok {
		return
	}
	if _, ok := msg.(cursor.BlinkMsg); ok {
		return
	}
	d.send(msg, remaining)
}

type listViewStyles interface {
	~struct {
		Title       lipgloss.Style
		Subtitle    lipgloss.Style
		Muted       lipgloss.Style
		Selected    lipgloss.Style
		ListPane    lipgloss.Style
		DetailPane  lipgloss.Style
		ErrorText   lipgloss.Style
		WarningText lipgloss.Style
	}
}

type listViewKeys interface {
	~struct {
		Up          key.Binding
		Down        key.Binding
		Enter       key.Binding
		Refresh     key.Binding
		Back        key.Binding
		Create      key.Binding
		Edit        key.Binding
		Delete      key.Binding
		Tags        key.Binding
		Adjust      key.Binding
		Set         key.Binding
		Publish     key.Binding
		Draft       key.Binding
		Complete    key.Binding
		CancelOrder key.Binding
	}
}

// DefaultListViewStyles returns a minimal style set for view model tests.
func DefaultListViewStyles[T listViewStyles]() T {
	return T{
		Title:       lipgloss.NewStyle(),
		Subtitle:    lipgloss.NewStyle(),
		Muted:       lipgloss.NewStyle(),
		Selected:    lipgloss.NewStyle(),
		ListPane:    lipgloss.NewStyle(),
		DetailPane:  lipgloss.NewStyle(),
		ErrorText:   lipgloss.NewStyle(),
		WarningText: lipgloss.NewStyle(),
	}
}

// DefaultListViewKeys returns key bindings for view model tests.
func DefaultListViewKeys[T listViewKeys]() T {
	return T{
		Up:          key.NewBinding(key.WithKeys(keyname.Up)),
		Down:        key.NewBinding(key.WithKeys(keyname.Down)),
		Enter:       key.NewBinding(key.WithKeys(keyname.Enter)),
		Refresh:     key.NewBinding(key.WithKeys("r")),
		Back:        key.NewBinding(key.WithKeys(keyname.Escape)),
		Create:      key.NewBinding(key.WithKeys("c")),
		Edit:        key.NewBinding(key.WithKeys("e")),
		Delete:      key.NewBinding(key.WithKeys("d")),
		Tags:        key.NewBinding(key.WithKeys("t")),
		Adjust:      key.NewBinding(key.WithKeys("a")),
		Set:         key.NewBinding(key.WithKeys("s")),
		Publish:     key.NewBinding(key.WithKeys("p")),
		Draft:       key.NewBinding(key.WithKeys("u")),
		Complete:    key.NewBinding(key.WithKeys("o")),
		CancelOrder: key.NewBinding(key.WithKeys("x")),
	}
}

// InitAndLoad runs Init and processes the resulting commands.
func InitAndLoad(t testing.TB, model views.ViewModel) views.ViewModel {
	t.Helper()
	cmd := model.Init()
	msgs := RunCmds(cmd)
	for _, msg := range msgs {
		updated, _ := model.Update(msg)
		model = updated
	}
	return model
}

// RunCmds executes a tea.Cmd and flattens any batch messages.
func RunCmds(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if msg == nil {
		return nil
	}
	switch typed := msg.(type) {
	case tea.BatchMsg:
		var out []tea.Msg
		for _, sub := range typed {
			out = append(out, RunCmds(sub)...)
		}
		return out
	default:
		return []tea.Msg{typed}
	}
}
