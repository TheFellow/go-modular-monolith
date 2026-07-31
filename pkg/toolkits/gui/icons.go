package gui

import (
	framework "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

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
	case IconAdd:
		return theme.ContentAddIcon()
	case IconRefresh:
		return theme.ViewRefreshIcon()
	case IconSave:
		return theme.DocumentSaveIcon()
	case IconCancel:
		return theme.CancelIcon()
	case IconBack, IconPrevious:
		return theme.NavigateBackIcon()
	case IconNext:
		return theme.NavigateNextIcon()
	case IconDelete:
		return theme.DeleteIcon()
	case IconTag:
		return theme.InfoIcon()
	case IconDashboard:
		return theme.HomeIcon()
	case IconDrinks:
		return theme.MediaMusicIcon()
	case IconIngredients:
		return theme.GridIcon()
	case IconInventory:
		return theme.StorageIcon()
	case IconMenus:
		return theme.DocumentIcon()
	case IconOrders:
		return theme.MailComposeIcon()
	case IconAudit:
		return theme.HistoryIcon()
	case IconTags:
		return theme.InfoIcon()
	case IconEmpty:
		return theme.SearchIcon()
	case IconCopy:
		return theme.ContentCopyIcon()
	default:
		return nil
	}
}

func WithIcon(button *SemanticButton, icon Icon) *SemanticButton {
	button.Icon = IconResource(icon)
	return button
}
