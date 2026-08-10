package store

import "context"

type Context interface {
	context.Context
	Transaction() (*Tx, bool)
}
