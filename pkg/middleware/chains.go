package middleware

import (
	"github.com/TheFellow/go-modular-monolith/pkg/uow"
)

var (
	Query   = NewQueryChain(QueryAuthZ())
	Command = NewCommandChain(CommandAuthZ(), UnitOfWork(uow.NewManager()))
)
