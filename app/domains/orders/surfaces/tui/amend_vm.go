package tui

import (
	"fmt"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	presentation "github.com/TheFellow/go-modular-monolith/app/domains/orders/surfaces"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/forms"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/keys"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/styles"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type amendVM struct {
	form                            *forms.Form
	draft                           presentation.AmendmentForm
	reason                          *forms.TextField
	replacements, ratios, garnishes []*forms.TextField
	steps                           []*stepsField
	viewport                        viewport.Model
	width, height                   int
	err                             error
	saving                          bool
}

func newAmendVM(order models.Order) *amendVM {
	v := &amendVM{draft: presentation.NewAmendmentForm(order), viewport: viewport.New(80, 20)}
	v.reason = forms.NewTextField("Reason", forms.WithRequired())
	fields := []forms.Field{v.reason}
	for _, field := range v.draft.Replacements {
		replacement := forms.NewTextField("Replace "+field.Name+" ("+field.ID.String()+")", forms.WithPlaceholder("Replacement ingredient ID; blank keeps current"))
		ratio := forms.NewTextField(field.Name+" quantity ratio", forms.WithInitialValue("1"))
		v.replacements, v.ratios = append(v.replacements, replacement), append(v.ratios, ratio)
		fields = append(fields, replacement, ratio)
	}
	for _, field := range v.draft.Preparation {
		steps := newStepsField(field.Name+" steps (one per line; Tab accepts)", field.Steps)
		garnish := forms.NewTextField(field.Name+" garnish (blank removes)", forms.WithInitialValue(field.Garnish))
		v.steps, v.garnishes = append(v.steps, steps), append(v.garnishes, garnish)
		fields = append(fields, steps, garnish)
	}
	v.form = forms.New(styles.Standard.Form, keys.Standard.Form, fields...)
	return v
}
func (v *amendVM) Init() tea.Cmd { return v.form.Init() }
func (v *amendVM) SetSize(width, height int) {
	v.width, v.height = width, height
	v.form.SetWidth(max(20, width-8))
	v.viewport.Width, v.viewport.Height = max(20, width), max(3, height-7)
}
func (v *amendVM) Update(msg tea.Msg) tea.Cmd {
	if v.saving {
		return nil
	}
	var cmd tea.Cmd
	v.form, cmd = v.form.Update(msg)
	return cmd
}
func (v *amendVM) Request() (models.Amendment, error) {
	v.draft.Reason = fmt.Sprint(v.reason.Value())
	for i := range v.draft.Replacements {
		v.draft.Replacements[i].ReplacementID = fmt.Sprint(v.replacements[i].Value())
		v.draft.Replacements[i].Ratio = fmt.Sprint(v.ratios[i].Value())
	}
	for i := range v.draft.Preparation {
		v.draft.Preparation[i].Steps = fmt.Sprint(v.steps[i].Value())
		v.draft.Preparation[i].Garnish = fmt.Sprint(v.garnishes[i].Value())
	}
	request, err := v.draft.Request()
	v.err = err
	return request, err
}
func (v *amendVM) View() string {
	content := v.form.View()
	v.viewport.SetContent(content)
	// Keep the actively selected field on screen even for large recipes.
	if field := v.form.FocusedField(); field != nil {
		before, _, found := strings.Cut(content, field.View())
		if found {
			y := strings.Count(before, "\n")
			if y < v.viewport.YOffset {
				v.viewport.SetYOffset(y)
			}
			fieldHeight := min(lipgloss.Height(field.View()), v.viewport.Height)
			if y+fieldHeight > v.viewport.YOffset+v.viewport.Height {
				v.viewport.SetYOffset(y + fieldHeight - v.viewport.Height)
			}
		}
	}
	message := ""
	if v.err != nil {
		message = "Error: " + v.err.Error()
	}
	if v.saving {
		message = "Approving amendment…"
	}
	header := fmt.Sprintf("Amend order %s · revision %d\nOriginal acceptance is preserved.\n%s", v.draft.OrderID, v.draft.Revision, message)
	return lipgloss.JoinVertical(lipgloss.Left, header, v.viewport.View(), "ctrl+s approve · ctrl+b add to batch · esc cancel", "tab next field · enter edit / accept (new line in steps)")
}

type amendmentsSavedMsg struct {
	err   error
	batch bool
}
