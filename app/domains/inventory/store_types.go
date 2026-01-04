package inventory

import "github.com/TheFellow/go-modular-monolith/app/domains/inventory/internal/dao"

func StoreTypes() []any {
	return dao.Types()
}
