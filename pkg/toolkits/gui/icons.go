package gui

import (
	"embed"
	"fmt"
	"image/color"
	"strings"
	"sync"

	framework "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

//go:embed icons/*.svg
var iconAssets embed.FS
var iconResources sync.Map

func themedIcon(name string) framework.Resource {
	if resource, ok := iconResources.Load(name); ok {
		return resource.(framework.Resource)
	}
	data, err := iconAssets.ReadFile("icons/" + name + ".svg")
	if err != nil {
		panic("missing embedded GUI icon: " + name)
	}
	var candidate framework.Resource
	if strings.HasPrefix(name, "lucide-") {
		candidate = &strokeThemedResource{name: name + ".svg", data: data}
	} else {
		candidate = theme.NewThemedResource(framework.NewStaticResource(name+".svg", data))
	}
	resource, _ := iconResources.LoadOrStore(name, candidate)
	return resource.(framework.Resource)
}

// strokeThemedResource preserves Lucide's fill="none" line art. Fyne's
// ThemedResource colorizer is intentionally fill-based and would otherwise
// turn every closed Lucide outline into a solid shape.
type strokeThemedResource struct {
	name string
	data []byte
}

func (r *strokeThemedResource) Name() string { return r.name }
func (r *strokeThemedResource) Content() []byte {
	foreground := color.NRGBAModel.Convert(theme.Color(theme.ColorNameForeground)).(color.NRGBA)
	hex := fmt.Sprintf("#%02x%02x%02x", foreground.R, foreground.G, foreground.B)
	return []byte(strings.ReplaceAll(string(r.data), "currentColor", hex))
}

// Icon names are presentation vocabulary, deliberately independent of labels
// and semantic IDs so visual choices can be reviewed and swapped centrally.
type Icon uint8

const (
	IconNone Icon = iota
	IconAdd
	IconRefresh
	IconSave
	IconCancel
	IconBack
	IconDelete
	IconTag
	IconPrevious
	IconNext
	IconDashboard
	IconDrinks
	IconIngredients
	IconInventory
	IconMenus
	IconOrders
	IconAudit
	IconTags
	IconEmpty
	IconCopy
)

func IconResource(icon Icon) framework.Resource {
	switch icon {
	case IconNone:
		return nil
	case IconAdd:
		return themedIcon("lucide-plus")
	case IconRefresh:
		return themedIcon("lucide-refresh-cw")
	case IconSave:
		return themedIcon("lucide-save")
	case IconCancel:
		return themedIcon("lucide-x")
	case IconBack, IconPrevious:
		return themedIcon("lucide-arrow-left")
	case IconNext:
		return themedIcon("lucide-arrow-right")
	case IconDelete:
		return themedIcon("lucide-trash-2")
	case IconTag:
		return themedIcon("lucide-tag")
	case IconDashboard:
		return themedIcon("lucide-layout-dashboard")
	case IconDrinks:
		return themedIcon("fontawesome-martini-glass-citrus")
	case IconIngredients:
		return themedIcon("lucide-carrot")
	case IconInventory:
		return themedIcon("lucide-warehouse")
	case IconMenus:
		return themedIcon("lucide-book-open-text")
	case IconOrders:
		return themedIcon("lucide-receipt-text")
	case IconAudit:
		return themedIcon("lucide-scroll-text")
	case IconTags:
		return themedIcon("lucide-tags")
	case IconEmpty:
		return themedIcon("lucide-search-x")
	case IconCopy:
		return themedIcon("lucide-copy")
	}
	return nil
}

func WithIcon(button *SemanticButton, icon Icon) *SemanticButton {
	button.Icon = IconResource(icon)
	return button
}
