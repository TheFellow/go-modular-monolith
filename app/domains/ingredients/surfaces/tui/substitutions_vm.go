package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	toolkit "github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/forms"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/keys"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/styles"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type substitutionsLoadedMsg struct {
	names map[entity.IngredientID]string
	owner *substitutionsVM
	rules []models.SubstitutionRule
	err   error
}
type substitutionSavedMsg struct {
	owner *substitutionsVM
	err   error
}

type substitutionsVM struct {
	names                    map[entity.IngredientID]string
	viewport                 toolkit.FormViewport
	fields                   []forms.Field
	app                      *app.Session
	ingredient               models.Ingredient
	writable                 bool
	rules                    []models.SubstitutionRule
	selected, width, height  int
	loading, submitting      bool
	err                      error
	form                     *forms.Form
	substitute, ratio, notes *forms.TextField
	quality, status          *forms.SelectField
	editing                  models.SubstitutionRule
}

func newSubstitutionsVM(session *app.Session, ingredient models.Ingredient, writable bool) *substitutionsVM {
	return &substitutionsVM{viewport: toolkit.NewFormViewport(), app: session, ingredient: ingredient, writable: writable}
}
func (m *substitutionsVM) load() tea.Cmd {
	if m.loading || m.submitting {
		return nil
	}
	m.loading, m.err = true, nil
	return func() tea.Msg {
		rules, err := m.app.Ingredients.SubstitutionRules(m.app.Context(), m.ingredient.ID)
		names := make(map[entity.IngredientID]string, len(rules))
		for _, rule := range rules {
			names[rule.SubstituteID] = rule.SubstituteID.String()
			if substitute, getErr := m.app.Ingredients.Get(m.app.Context(), rule.SubstituteID); getErr == nil {
				names[rule.SubstituteID] = substitute.Name
			}
		}
		return substitutionsLoadedMsg{owner: m, rules: rules, names: names, err: err}
	}
}
func (m *substitutionsVM) setSize(width, height int) {
	m.width, m.height = width, height
	m.viewport.SetSize(width, height)
	if m.form != nil {
		m.form.SetWidth(max(1, width-2))
	}
}
func (m *substitutionsVM) start(rule models.SubstitutionRule) tea.Cmd {
	m.editing = rule
	id := ""
	if !rule.SubstituteID.IsZero() {
		id = rule.SubstituteID.String()
	}
	m.substitute = forms.NewTextField("Substitute ingredient ID", forms.WithInitialValue(id), forms.WithRequired())
	m.ratio = forms.NewTextField("Ratio", forms.WithInitialValue(strconv.FormatFloat(rule.Ratio, 'g', -1, 64)), forms.WithRequired())
	m.notes = forms.NewTextField("Notes", forms.WithInitialValue(rule.Notes))
	m.quality = forms.NewSelectField("Quality impact", []forms.SelectOption{{Label: "equivalent", Value: models.QualityEquivalent}, {Label: "similar", Value: models.QualitySimilar}, {Label: "different", Value: models.QualityDifferent}}, forms.WithInitialValue(rule.QualityImpact))
	m.status = forms.NewSelectField("Status", []forms.SelectOption{{Label: "Enabled", Value: false}, {Label: "Disabled", Value: true}}, forms.WithInitialValue(rule.Disabled))
	fields := []forms.Field{m.ratio, m.quality, m.notes, m.status}
	if rule.Revision == 0 {
		fields = append([]forms.Field{m.substitute}, fields...)
	}
	m.fields = fields
	m.form = forms.New(styles.Standard.Form, keys.Standard.Form, fields...)
	m.form.SetWidth(max(1, m.width-2))
	m.err = nil
	return m.form.Init()
}
func (m *substitutionsVM) update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case substitutionsLoadedMsg:
		if msg.owner != m {
			return nil
		}
		m.loading, m.err = false, msg.err
		if msg.err == nil {
			m.rules, m.names = msg.rules, msg.names
			if m.selected >= len(m.rules) {
				m.selected = max(0, len(m.rules)-1)
			}
		}
		return nil
	case substitutionSavedMsg:
		if msg.owner != m {
			return nil
		}
		m.submitting, m.err = false, msg.err
		if msg.err == nil {
			m.form = nil
			return m.load()
		}
		return nil
	case tea.KeyMsg:
		if m.submitting {
			return nil
		}
		if m.form != nil {
			if msg.String() == "esc" && !m.form.IsEditing() {
				m.form = nil
				m.err = nil
				return nil
			}
			if key.Matches(msg, keys.Standard.Form.Submit) {
				return m.submit()
			}
		} else {
			switch msg.String() {
			case "j", "down":
				m.selected = min(max(0, len(m.rules)-1), m.selected+1)
			case "k", "up":
				m.selected = max(0, m.selected-1)
			case "r":
				return m.load()
			case "c", "n":
				if m.writable && !m.loading {
					return m.start(models.SubstitutionRule{Ratio: 1, QualityImpact: models.QualityEquivalent})
				}
			case "enter", "e":
				if m.writable && !m.loading && len(m.rules) > 0 {
					return m.start(m.rules[m.selected])
				}
			}
			return nil
		}
	}
	if m.form != nil {
		var cmd tea.Cmd
		m.form, cmd = m.form.Update(msg)
		return cmd
	}
	return nil
}
func (m *substitutionsVM) submit() tea.Cmd {
	if !m.writable || m.submitting {
		return nil
	}
	if err := m.form.Validate(); err != nil {
		m.err = err
		return nil
	}
	rule := m.editing
	rule.IngredientID = m.ingredient.ID
	var err error
	if rule.Revision == 0 {
		rule.SubstituteID, err = entity.ParseIngredientID(strings.TrimSpace(toString(m.substitute.Value())))
	}
	if err != nil {
		m.err = err
		return nil
	}
	rule.Ratio, err = strconv.ParseFloat(strings.TrimSpace(toString(m.ratio.Value())), 64)
	if err != nil {
		m.err = errors.Invalidf("ratio must be a positive number")
		return nil
	}
	rule.QualityImpact, _ = m.quality.Value().(models.Quality)
	rule.Disabled, _ = m.status.Value().(bool)
	rule.Notes = strings.TrimSpace(toString(m.notes.Value()))
	if err := rule.Validate(); err != nil {
		m.err = err
		return nil
	}
	m.submitting, m.err = true, nil
	return func() tea.Msg {
		_, err := m.app.Ingredients.SetSubstitution(m.app.Context(), &rule)
		return substitutionSavedMsg{owner: m, err: err}
	}
}
func (m *substitutionsVM) view() string {
	lines := []string{styles.Standard.ListView.Title.Render("Substitutions · " + m.ingredient.Name)}
	if m.err != nil {
		lines = append(lines, styles.Standard.Form.Error.Render("Error: "+m.err.Error()))
	}
	if m.loading {
		lines = append(lines, "Loading rules…")
	}
	if m.submitting {
		lines = append(lines, "Saving rule…")
	}
	if m.form != nil {
		if m.editing.Revision > 0 {
			lines = append(lines, fmt.Sprintf("Substitute: %s · revision %d", m.editing.SubstituteID, m.editing.Revision))
		}
		lines = append(lines, m.form.View(), "After a conflict, return to the list, refresh, and select the current rule.")
	} else {
		lines = append(lines, fmt.Sprintf("%d rules (including disabled)", len(m.rules)), "")
		pageSize := max(1, m.height-9)
		first := m.selected / pageSize * pageSize
		for i := first; i < min(len(m.rules), first+pageSize); i++ {
			rule := m.rules[i]
			state := "enabled"
			if rule.Disabled {
				state = "disabled"
			}
			prefix := "  "
			if i == m.selected {
				prefix = "> "
			}
			lines = append(lines, fmt.Sprintf("%s%s · ratio %g · %s · %s · revision %d", prefix, m.names[rule.SubstituteID], rule.Ratio, rule.QualityImpact, state, rule.Revision))
		}
		if len(m.rules) == 0 && !m.loading {
			lines = append(lines, "No substitution rules.")
		}
		if len(m.rules) > 0 {
			lines = append(lines, "", "Substitute ID: "+m.rules[m.selected].SubstituteID.String(), "Notes: "+m.rules[m.selected].Notes)
		}
	}
	content := strings.Join(lines, "\n")
	focusLine := 0
	if m.form != nil {
		focusLine = 2
		for _, field := range m.fields {
			if field == m.form.FocusedField() {
				break
			}
			focusLine += strings.Count(field.View(), "\n") + 2
		}
	}
	if m.width > 0 {
		content = lipgloss.NewStyle().Width(m.width).Render(content)
	}
	return m.viewport.View(content, focusLine, "")
}
