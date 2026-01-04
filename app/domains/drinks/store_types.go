package drinks

import "github.com/TheFellow/go-modular-monolith/app/domains/drinks/internal/dao"

func StoreTypes() []any {
	return dao.Types()
}
