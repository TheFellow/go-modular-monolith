package tui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app"
	menus "github.com/TheFellow/go-modular-monolith/app/domains/menus"
	menumodels "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/paging"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type placeMenu struct {
	id     entity.MenuID
	name   string
	drinks []placeDrink
}
type placeDrink struct {
	id   entity.DrinkID
	name string
}
type placeLine struct {
	drink    placeDrink
	quantity int
	notes    string
}
type placeCatalogLoadedMsg struct {
	workflow uint64
	menus    []placeMenu
	err      error
}
type orderPlacedMsg struct {
	workflow uint64
	order    *models.Order
	err      error
}

type placeVM struct {
	session                               *app.Session
	workflow                              uint64
	menus, visibleMenus                   []placeMenu
	drinks, visibleDrinks                 []placeDrink
	menuIndex, drinkIndex, focus          int
	menuQuery, drinkQuery, quantity, tags textinput.Model
	tagsDirty                             bool
	itemNotes, orderNotes                 textarea.Model
	menu                                  *placeMenu
	lines                                 []placeLine
	loading, saving                       bool
	err                                   error
	backArmed                             bool
}

func newPlaceVM(session *app.Session, workflow uint64) *placeVM {
	input := func(prompt string) textinput.Model { v := textinput.New(); v.Prompt = prompt; return v }
	notes := func(placeholder string) textarea.Model {
		v := textarea.New()
		v.Placeholder = placeholder
		v.SetHeight(3)
		v.ShowLineNumbers = false
		return v
	}
	v := &placeVM{session: session, workflow: workflow, loading: true,
		menuQuery: input("Search menus: "), drinkQuery: input("Search drinks: "), quantity: input("Quantity: "),
		tags:      input("Complete tags (optional): "),
		itemNotes: notes("Item notes"), orderNotes: notes("Order notes")}
	v.quantity.SetValue("1")
	v.menuQuery.Focus()
	return v
}
func (v *placeVM) Init() tea.Cmd { return v.loadCatalog() }
func (v *placeVM) loadCatalog() tea.Cmd {
	workflow := v.workflow
	session := v.session
	return func() tea.Msg {
		all, err := paging.Collect(func(cursor paging.Cursor) (paging.Page[*menumodels.Menu], error) {
			return session.Menus.List(session.Context(), menus.ListRequest{Status: menumodels.MenuStatusPublished, Cursor: cursor})
		})
		if err != nil {
			return placeCatalogLoadedMsg{workflow: workflow, err: err}
		}
		result := make([]placeMenu, 0, len(all))
		for _, menu := range all {
			if menu == nil {
				return placeCatalogLoadedMsg{workflow: workflow, err: errors.Internalf("menu missing")}
			}
			entry := placeMenu{id: menu.ID, name: menu.Name}
			for _, item := range menu.Items {
				if item.Availability == menumodels.AvailabilityUnavailable {
					continue
				}
				name, ok := item.DisplayName.Unwrap()
				if !ok || strings.TrimSpace(name) == "" {
					drink, getErr := session.Drinks.Get(session.Context(), item.DrinkID)
					if getErr != nil {
						return placeCatalogLoadedMsg{workflow: workflow, err: getErr}
					}
					if drink == nil {
						return placeCatalogLoadedMsg{workflow: workflow, err: errors.Internalf("drink %s missing", item.DrinkID)}
					}
					name = drink.Name
				}
				entry.drinks = append(entry.drinks, placeDrink{id: item.DrinkID, name: name})
			}
			sort.Slice(entry.drinks, func(i, j int) bool {
				if strings.EqualFold(entry.drinks[i].name, entry.drinks[j].name) {
					return entry.drinks[i].id.String() < entry.drinks[j].id.String()
				}
				return strings.ToLower(entry.drinks[i].name) < strings.ToLower(entry.drinks[j].name)
			})
			result = append(result, entry)
		}
		sort.Slice(result, func(i, j int) bool {
			if strings.EqualFold(result[i].name, result[j].name) {
				return result[i].id.String() < result[j].id.String()
			}
			return strings.ToLower(result[i].name) < strings.ToLower(result[j].name)
		})
		return placeCatalogLoadedMsg{workflow: workflow, menus: result}
	}
}
func (v *placeVM) setCatalog(msg placeCatalogLoadedMsg) {
	v.loading, v.err, v.menus = false, msg.err, msg.menus
	v.filterMenus()
}
func (v *placeVM) filterMenus() {
	q := strings.ToLower(strings.TrimSpace(v.menuQuery.Value()))
	v.visibleMenus = nil
	for _, m := range v.menus {
		if q == "" || strings.Contains(strings.ToLower(m.name), q) {
			v.visibleMenus = append(v.visibleMenus, m)
		}
	}
	if v.menuIndex >= len(v.visibleMenus) {
		v.menuIndex = max(0, len(v.visibleMenus)-1)
	}
}
func (v *placeVM) filterDrinks() {
	q := strings.ToLower(strings.TrimSpace(v.drinkQuery.Value()))
	v.visibleDrinks = nil
	for _, d := range v.drinks {
		if q == "" || strings.Contains(strings.ToLower(d.name), q) {
			v.visibleDrinks = append(v.visibleDrinks, d)
		}
	}
	if v.drinkIndex >= len(v.visibleDrinks) {
		v.drinkIndex = max(0, len(v.visibleDrinks)-1)
	}
}
func (v *placeVM) chooseMenu() {
	if v.menuIndex < 0 || v.menuIndex >= len(v.visibleMenus) {
		return
	}
	picked := v.visibleMenus[v.menuIndex]
	v.menu = &picked
	v.drinks = append([]placeDrink(nil), picked.drinks...)
	v.lines = nil
	v.filterDrinks()
	v.focus = 1
	v.refocus()
}
func (v *placeVM) add() {
	if v.menu == nil || v.drinkIndex < 0 || v.drinkIndex >= len(v.visibleDrinks) {
		v.err = errors.Invalidf("available menu drink is required")
		return
	}
	qty, err := strconv.Atoi(strings.TrimSpace(v.quantity.Value()))
	if err != nil || qty <= 0 {
		v.err = errors.Invalidf("quantity must be > 0")
		return
	}
	d := v.visibleDrinks[v.drinkIndex]
	notes := strings.TrimSpace(v.itemNotes.Value())
	for i := range v.lines {
		if v.lines[i].drink.id == d.id {
			v.lines[i].quantity += qty
			v.lines[i].notes = notes
			v.err = nil
			return
		}
	}
	v.lines = append(v.lines, placeLine{drink: d, quantity: qty, notes: notes})
	v.quantity.SetValue("1")
	v.itemNotes.SetValue("")
	v.err = nil
}
func (v *placeVM) submit() tea.Cmd {
	if v.saving {
		return nil
	}
	if v.menu == nil {
		v.err = errors.Invalidf("menu is required")
		return nil
	}
	if len(v.lines) == 0 {
		v.err = errors.Invalidf("order must have at least 1 item")
		return nil
	}
	order := &models.Order{MenuID: v.menu.id, Notes: strings.TrimSpace(v.orderNotes.Value())}
	for _, line := range v.lines {
		order.Items = append(order.Items, models.OrderItem{DrinkID: line.drink.id, Quantity: line.quantity, Notes: line.notes})
	}
	var desired *tag.Tags
	if v.tagsDirty {
		values, err := tag.ParseCollection(strings.TrimSpace(v.tags.Value()))
		if err != nil {
			v.err = errors.Invalidf("invalid tags: %v", err)
			return nil
		}
		desired = &values
	}
	v.saving = true
	workflow := v.workflow
	session := v.session
	return func() tea.Msg {
		placed, err := app.RunTaggedMutation(session.App, session.Context(), desired, func(ctx *middleware.Context) (*models.Order, error) {
			return session.Orders.Place(ctx, order)
		})
		return orderPlacedMsg{workflow: workflow, order: placed, err: err}
	}
}
func (v *placeVM) dirty() bool {
	return v.menu != nil || len(v.lines) > 0 || strings.TrimSpace(v.menuQuery.Value()) != "" || strings.TrimSpace(v.drinkQuery.Value()) != "" || strings.TrimSpace(v.itemNotes.Value()) != "" || strings.TrimSpace(v.orderNotes.Value()) != "" || v.tagsDirty
}
func (v *placeVM) mayClose() bool {
	if !v.dirty() || v.backArmed {
		return true
	}
	v.backArmed = true
	v.err = errors.Invalidf("unsaved order; press esc again to discard")
	return false
}
func (v *placeVM) refocus() {
	v.menuQuery.Blur()
	v.drinkQuery.Blur()
	v.quantity.Blur()
	v.itemNotes.Blur()
	v.orderNotes.Blur()
	v.tags.Blur()
	switch v.focus {
	case 0:
		v.menuQuery.Focus()
	case 1:
		v.drinkQuery.Focus()
	case 2:
		v.quantity.Focus()
	case 3:
		v.itemNotes.Focus()
	case 4:
		v.orderNotes.Focus()
	case 5:
		v.tags.Focus()
	}
}
func (v *placeVM) Update(msg tea.Msg) tea.Cmd {
	key, ok := msg.(tea.KeyMsg)
	if ok {
		v.backArmed = false
		switch key.String() {
		case "tab":
			v.focus = (v.focus + 1) % 6
			v.refocus()
			return nil
		case "shift+tab":
			v.focus = (v.focus + 5) % 6
			v.refocus()
			return nil
		case "up":
			if v.focus == 0 && v.menuIndex > 0 {
				v.menuIndex--
			}
			if v.focus == 1 && v.drinkIndex > 0 {
				v.drinkIndex--
			}
			return nil
		case "down":
			if v.focus == 0 && v.menuIndex+1 < len(v.visibleMenus) {
				v.menuIndex++
			}
			if v.focus == 1 && v.drinkIndex+1 < len(v.visibleDrinks) {
				v.drinkIndex++
			}
			return nil
		case "enter":
			if v.focus == 0 {
				v.chooseMenu()
				return nil
			}
			if v.focus == 1 {
				v.add()
				return nil
			}
		}
	}
	var cmd tea.Cmd
	switch v.focus {
	case 0:
		v.menuQuery, cmd = v.menuQuery.Update(msg)
		v.filterMenus()
	case 1:
		v.drinkQuery, cmd = v.drinkQuery.Update(msg)
		v.filterDrinks()
	case 2:
		v.quantity, cmd = v.quantity.Update(msg)
	case 3:
		v.itemNotes, cmd = v.itemNotes.Update(msg)
	case 4:
		v.orderNotes, cmd = v.orderNotes.Update(msg)
	case 5:
		before := v.tags.Value()
		v.tags, cmd = v.tags.Update(msg)
		v.tagsDirty = v.tagsDirty || v.tags.Value() != before
	}
	return cmd
}
func (v *placeVM) View() string {
	lines := []string{"Place Order", "", v.menuQuery.View()}
	if v.loading {
		lines = append(lines, "Loading published menus...")
	} else {
		for i, m := range v.visibleMenus {
			prefix := "  "
			if i == v.menuIndex {
				prefix = "> "
			}
			lines = append(lines, prefix+m.name)
		}
	}
	if v.menu != nil {
		lines = append(lines, "", "Menu: "+v.menu.name, v.drinkQuery.View())
		for i, d := range v.visibleDrinks {
			prefix := "  "
			if i == v.drinkIndex {
				prefix = "> "
			}
			lines = append(lines, prefix+d.name)
		}
		lines = append(lines, "", v.quantity.View(), "Item notes:", v.itemNotes.View(), "enter on a drink adds it")
		for _, line := range v.lines {
			lines = append(lines, fmt.Sprintf("• %s × %d — %s", line.drink.name, line.quantity, line.notes))
		}
		lines = append(lines, "", "Order notes:", v.orderNotes.View(), "", v.tags.View())
	}
	if v.saving {
		lines = append(lines, "", "Saving...")
	}
	if v.err != nil {
		lines = append(lines, "", "Error: "+v.err.Error())
	}
	lines = append(lines, "", "ctrl+s place • esc back")
	return strings.Join(lines, "\n")
}
