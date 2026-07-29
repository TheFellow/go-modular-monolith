// Package fyne contains reusable desktop presentation mechanics for Fyne
// applications. It deliberately has no knowledge of Mixology or its domains.
package fyne

import (
	"fmt"

	framework "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// View is a framework-native presentation supplied by an application surface.
// Views own their widgets and state; Shell owns only top-level navigation.
type View interface {
	Title() string
	Content() framework.CanvasObject
}

// Activated is an optional view lifecycle implemented by presentations that
// need to refresh whenever they are selected.
type Activated interface {
	Activate()
}

// Route describes one lazily constructed top-level view.
type Route struct {
	ID    string
	Label string
	Build func() View
}

// Shell coordinates navigation and preserves each constructed view for the
// lifetime of the window.
type Shell struct {
	routes           map[string]Route
	order            []string
	views            map[string]View
	current          string
	title            *widget.Label
	body             *framework.Container
	content          framework.CanvasObject
	initialActivated bool
}

// NewShell constructs an application shell and selects initialRoute.
func NewShell(routes []Route, initialRoute string) (*Shell, error) {
	if len(routes) == 0 {
		return nil, fmt.Errorf("fyne shell requires at least one route")
	}

	s := &Shell{
		routes: make(map[string]Route, len(routes)),
		views:  make(map[string]View, len(routes)),
		title:  widget.NewLabel(""),
		body:   container.NewStack(),
	}
	s.title.TextStyle = framework.TextStyle{Bold: true}

	buttons := make([]framework.CanvasObject, 0, len(routes)+1)
	buttons = append(buttons, widget.NewLabelWithStyle("Mixology", framework.TextAlignLeading, framework.TextStyle{Bold: true}))
	for _, route := range routes {
		if route.ID == "" || route.Label == "" || route.Build == nil {
			return nil, fmt.Errorf("fyne shell route must have id, label, and builder")
		}
		if _, exists := s.routes[route.ID]; exists {
			return nil, fmt.Errorf("duplicate fyne shell route %q", route.ID)
		}
		s.routes[route.ID] = route
		s.order = append(s.order, route.ID)
		id := route.ID
		buttons = append(buttons, widget.NewButton(route.Label, func() { _ = s.Navigate(id) }))
	}

	navigation := container.NewVBox(buttons...)
	s.content = container.NewBorder(
		container.NewPadded(s.title), nil,
		container.NewPadded(container.NewVBox(navigation, layout.NewSpacer())), nil,
		container.NewPadded(s.body),
	)
	if err := s.navigate(initialRoute, false); err != nil {
		return nil, err
	}
	return s, nil
}

// Content returns the shell's root canvas object.
func (s *Shell) Content() framework.CanvasObject { return s.content }

// Current returns the selected route ID.
func (s *Shell) Current() string { return s.current }

// RouteIDs returns the stable navigation order used by menus and shortcuts.
func (s *Shell) RouteIDs() []string { return append([]string(nil), s.order...) }

// RouteLabel returns the displayed label for a configured route.
func (s *Shell) RouteLabel(id string) string { return s.routes[id].Label }

// ExecuteCommand offers an application-shell intent to the selected surface.
func (s *Shell) ExecuteCommand(command Command) bool {
	view, ok := s.views[s.current]
	if !ok {
		return false
	}
	commander, ok := view.(Commander)
	return ok && commander.ExecuteCommand(command)
}

// Navigate selects a route, constructing its view on first use.
func (s *Shell) Navigate(id string) error {
	return s.navigate(id, true)
}

// ActivateCurrent activates the initial route after the application has
// attached Shell.Content to its window. Subsequent Navigate calls activate
// their selected route automatically.
func (s *Shell) ActivateCurrent() {
	if s.initialActivated {
		return
	}
	s.initialActivated = true
	if view, ok := s.views[s.current]; ok {
		if activated, ok := view.(Activated); ok {
			activated.Activate()
		}
	}
}

func (s *Shell) navigate(id string, activate bool) error {
	route, ok := s.routes[id]
	if !ok {
		return fmt.Errorf("unknown fyne shell route %q", id)
	}
	view, ok := s.views[id]
	if !ok {
		view = route.Build()
		if view == nil || view.Content() == nil {
			return fmt.Errorf("fyne shell route %q built an invalid view", id)
		}
		s.views[id] = view
	}
	s.current = id
	s.title.SetText(view.Title())
	s.body.Objects = []framework.CanvasObject{view.Content()}
	s.body.Refresh()
	if activate {
		s.initialActivated = true
	}
	if activated, ok := view.(Activated); activate && ok {
		activated.Activate()
	}
	return nil
}
