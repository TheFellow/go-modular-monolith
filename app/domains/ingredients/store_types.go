package ingredients

import "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/internal/dao"

func StoreTypes() []any {
	return dao.Types()
}
