package orders

import "github.com/TheFellow/go-modular-monolith/app/domains/orders/internal/dao"

func StoreTypes() []any {
	return dao.Types()
}
