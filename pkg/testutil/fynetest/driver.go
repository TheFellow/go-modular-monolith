package fynetest

import (
	"testing"

	framework "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	gui "github.com/TheFellow/go-modular-monolith/pkg/toolkits/gui"
)

// Driver locates semantic controls and interacts through Fyne's test driver.
type Driver struct {
	t    testing.TB
	root framework.CanvasObject
}

func NewDriver(t testing.TB, root framework.CanvasObject) Driver {
	t.Helper()
	return Driver{t: t, root: root}
}

func (d Driver) Tap(id string) {
	d.t.Helper()
	object := d.find(id)
	tappable, ok := object.(framework.Tappable)
	if !ok {
		d.t.Fatalf("semantic control %q is not tappable", id)
	}
	test.Tap(tappable)
}

func (d Driver) Type(id, value string) {
	d.t.Helper()
	object := d.find(id)
	entry, ok := object.(*gui.SemanticEntry)
	if !ok {
		d.t.Fatalf("semantic control %q is not an entry", id)
	}
	entry.SetText("")
	test.Type(entry, value)
}

// Submit invokes the entry's Enter-key action.
func (d Driver) Submit(id string) {
	d.t.Helper()
	object := d.find(id)
	entry, ok := object.(*gui.SemanticEntry)
	if !ok {
		d.t.Fatalf("semantic control %q is not an entry", id)
	}
	if entry.OnSubmitted != nil {
		entry.OnSubmitted(entry.Text)
	}
}

func (d Driver) find(id string) framework.CanvasObject {
	d.t.Helper()
	var found framework.CanvasObject
	walk(d.root, func(object framework.CanvasObject) bool {
		semantic, ok := object.(gui.Semantic)
		if ok && semantic.SemanticID() == id {
			found = object
			return false
		}
		return true
	})
	if found == nil {
		d.t.Fatalf("semantic control %q not found", id)
	}
	return found
}

func walk(object framework.CanvasObject, visit func(framework.CanvasObject) bool) {
	if object == nil || !visit(object) {
		return
	}
	switch object := object.(type) {
	case interface {
		SemanticChildren() []framework.CanvasObject
	}:
		for _, child := range object.SemanticChildren() {
			walk(child, visit)
		}
	case *framework.Container:
		for _, child := range object.Objects {
			walk(child, visit)
		}
	case *container.Scroll:
		walk(object.Content, visit)
	case *container.Split:
		walk(object.Leading, visit)
		walk(object.Trailing, visit)
	case *widget.Card:
		walk(object.Content, visit)
	}
}
