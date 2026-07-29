package tui

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app"
	drinks "github.com/TheFellow/go-modular-monolith/app/domains/drinks"
	drinkmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/queries"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/paging"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	addDrinkKey    = key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add drink"))
	removeDrinkKey = key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "remove drink"))
	analyzeKey     = key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "analyze"))
)

type drinkChoice struct {
	id   entity.DrinkID
	name string
}

type drinkPicker struct {
	query       textinput.Model
	tags        textinput.Model
	tagsDirty   bool
	editingTags bool
	all         []drinkChoice
	visible     []drinkChoice
	selected    int
	err         error
	loading     bool
	saving      bool
}

func newDrinkPicker(current tag.Tags) *drinkPicker {
	input := textinput.New()
	input.Placeholder = "Search drinks"
	input.Prompt = "> "
	input.Focus()
	tags := textinput.New()
	tags.Placeholder = "unchanged; edit to clear or replace"
	tags.SetValue(current.Canonical().String())
	return &drinkPicker{query: input, tags: tags, loading: true}
}

func (p *drinkPicker) setChoices(choices []drinkChoice, err error) {
	p.loading, p.err, p.all = false, err, choices
	p.filter()
}

func (p *drinkPicker) filter() {
	needle := strings.ToLower(strings.TrimSpace(p.query.Value()))
	p.visible = p.visible[:0]
	for _, choice := range p.all {
		if needle == "" || strings.Contains(strings.ToLower(choice.name), needle) {
			p.visible = append(p.visible, choice)
		}
	}
	if p.selected >= len(p.visible) {
		p.selected = max(0, len(p.visible)-1)
	}
}

func (p *drinkPicker) update(msg tea.Msg) tea.Cmd {
	if typed, ok := msg.(tea.KeyMsg); ok {
		if typed.Type == tea.KeyTab {
			p.editingTags = !p.editingTags
			if p.editingTags {
				p.query.Blur()
				return p.tags.Focus()
			}
			p.tags.Blur()
			return p.query.Focus()
		}
		if p.editingTags {
			before := p.tags.Value()
			var cmd tea.Cmd
			p.tags, cmd = p.tags.Update(msg)
			p.tagsDirty = p.tagsDirty || p.tags.Value() != before
			return cmd
		}
		switch typed.Type {
		case tea.KeyUp:
			if p.selected > 0 {
				p.selected--
			}
			return nil
		case tea.KeyDown:
			if p.selected+1 < len(p.visible) {
				p.selected++
			}
			return nil
		}
	}
	var cmd tea.Cmd
	p.query, cmd = p.query.Update(msg)
	if _, ok := msg.(tea.KeyMsg); ok {
		p.filter()
	}
	return cmd
}

func (p *drinkPicker) choice() (drinkChoice, bool) {
	if p.selected < 0 || p.selected >= len(p.visible) {
		return drinkChoice{}, false
	}
	return p.visible[p.selected], true
}

func (p *drinkPicker) view(title string) string {
	lines := []string{title, "", p.query.View(), "", "Complete tags (optional)", p.tags.View(), "tab switches search/tags", ""}
	if p.loading {
		return strings.Join(append(lines, "Loading drinks..."), "\n")
	}
	if p.saving {
		return strings.Join(append(lines, "Saving..."), "\n")
	}
	if p.err != nil {
		return strings.Join(append(lines, lipgloss.NewStyle().Width(100).Render("Error: "+p.err.Error())), "\n")
	}
	if len(p.visible) == 0 {
		return strings.Join(append(lines, "No matching drinks"), "\n")
	}
	end := min(len(p.visible), 20)
	start := 0
	if p.selected >= end {
		start = p.selected - 19
		end = min(len(p.visible), start+20)
	}
	for i := start; i < end; i++ {
		choice := p.visible[i]
		prefix := "  "
		if i == p.selected {
			prefix = "> "
		}
		lines = append(lines, prefix+choice.name+"  ·  "+choice.id.String())
	}
	if len(p.visible) > end-start {
		lines = append(lines, fmt.Sprintf("\nShowing %d-%d of %d", start+1, end, len(p.visible)))
	}
	return strings.Join(lines, "\n")
}

func (p *drinkPicker) desiredTags() (*tag.Tags, error) {
	if !p.tagsDirty {
		return nil, nil
	}
	values, err := tag.ParseCollection(strings.TrimSpace(p.tags.Value()))
	if err != nil {
		return nil, errors.Invalidf("invalid tags: %v", err)
	}
	return &values, nil
}

type analysisVM struct {
	input   textinput.Model
	result  *queries.MenuAnalytics
	err     error
	loading bool
}

func newAnalysisVM() *analysisVM {
	input := textinput.New()
	input.Prompt = "Target margin (0-1): "
	input.SetValue("0.70")
	input.Focus()
	return &analysisVM{input: input}
}

func (a *analysisVM) target() (float64, error) {
	target, err := strconv.ParseFloat(strings.TrimSpace(a.input.Value()), 64)
	if err != nil || math.IsNaN(target) || math.IsInf(target, 0) || target <= 0 || target >= 1 {
		return 0, errors.Invalidf("target margin must be a number between 0 and 1")
	}
	return target, nil
}

func (a *analysisVM) view() string {
	lines := []string{"Menu cost and availability analysis", "", a.input.View(), ""}
	if a.loading {
		lines = append(lines, "Analyzing...")
	}
	if a.err != nil {
		lines = append(lines, "Error: "+a.err.Error())
	}
	if a.result != nil {
		lines = append(lines, analysisText(*a.result))
	}
	return strings.Join(lines, "\n")
}

func analysisText(analysis queries.MenuAnalytics) string {
	lines := []string{fmt.Sprintf("Available: %d/%d", analysis.AvailableCount, analysis.TotalCount)}
	if analysis.AverageMargin != nil {
		lines = append(lines, fmt.Sprintf("Average margin: %.0f%%", *analysis.AverageMargin*100))
	}
	for _, item := range analysis.Items {
		cost := "unknown"
		if item.Cost != nil && !item.CostUnknown {
			cost = item.Cost.String()
		}
		price := "n/a"
		if item.MenuPrice != nil {
			price = item.MenuPrice.String()
		} else if item.SuggestedPrice != nil {
			price = "suggested " + item.SuggestedPrice.String()
		}
		margin := "n/a"
		if item.Margin != nil {
			margin = fmt.Sprintf("%.0f%%", *item.Margin*100)
		}
		status := strings.ToUpper(string(item.Availability))
		if len(item.Substitutions) > 0 {
			sub := item.Substitutions[0]
			status += fmt.Sprintf(" (sub: %s for %s)", sub.Substitute.String(), sub.Original.String())
		}
		lines = append(lines, fmt.Sprintf("\n%s\nID: %s\nCost: %s\nPrice: %s\nMargin: %s\nStatus: %s", item.Name, item.DrinkID.String(), cost, price, margin, status))
	}
	return strings.Join(lines, "\n")
}

func loadDrinkChoices(session *app.Session, menu models.Menu, removing bool, workflowID uint64) tea.Cmd {
	return func() tea.Msg {
		if removing {
			choices := make([]drinkChoice, 0, len(menu.Items))
			for _, item := range menu.Items {
				name, ok := item.DisplayName.Unwrap()
				if !ok || strings.TrimSpace(name) == "" {
					drink, err := session.Drinks.Get(session.Context(), item.DrinkID)
					if err != nil {
						return drinkChoicesLoadedMsg{workflowID: workflowID, err: err}
					}
					if drink == nil {
						return drinkChoicesLoadedMsg{workflowID: workflowID, err: errors.Internalf("drink %s missing", item.DrinkID.String())}
					}
					name = drink.Name
				}
				choices = append(choices, drinkChoice{id: item.DrinkID, name: name})
			}
			sort.Slice(choices, func(i, j int) bool { return choices[i].name < choices[j].name })
			return drinkChoicesLoadedMsg{workflowID: workflowID, choices: choices}
		}
		all, err := paging.Collect(func(cursor paging.Cursor) (paging.Page[*drinkmodels.Drink], error) {
			return session.Drinks.List(session.Context(), drinks.ListRequest{Cursor: cursor})
		})
		if err != nil {
			return drinkChoicesLoadedMsg{workflowID: workflowID, err: err}
		}
		included := make(map[entity.DrinkID]bool, len(menu.Items))
		for _, item := range menu.Items {
			included[item.DrinkID] = true
		}
		choices := make([]drinkChoice, 0, len(all))
		for _, drink := range all {
			if drink != nil && !included[drink.ID] {
				choices = append(choices, drinkChoice{id: drink.ID, name: drink.Name})
			}
		}
		sort.Slice(choices, func(i, j int) bool {
			if choices[i].name == choices[j].name {
				return choices[i].id.String() < choices[j].id.String()
			}
			return choices[i].name < choices[j].name
		})
		return drinkChoicesLoadedMsg{workflowID: workflowID, choices: choices}
	}
}

type drinkChoicesLoadedMsg struct {
	workflowID uint64
	choices    []drinkChoice
	err        error
}
type drinkAddedMsg struct {
	workflowID uint64
	err        error
}
type drinkRemovedMsg struct {
	workflowID uint64
	err        error
}
type analysisLoadedMsg struct {
	workflowID uint64
	value      queries.MenuAnalytics
	err        error
}
