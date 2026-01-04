package menu

import "github.com/TheFellow/go-modular-monolith/app/domains/menu/internal/dao"

func StoreTypes() []any {
	return dao.Types()
}
