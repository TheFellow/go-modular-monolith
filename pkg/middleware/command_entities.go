package middleware

import "context"

type commandLoader func(*Context) (CedarEntity, error)

type commandLoaderKey struct{}
type commandInputKey struct{}
type commandOutputKey struct{}

func WithCommandLoader(loader commandLoader) ContextOpt {
	return func(c *Context) {
		if loader == nil {
			return
		}
		c.Context = context.WithValue(c.Context, commandLoaderKey{}, loader)
	}
}

func commandLoaderFromContext(ctx context.Context) (commandLoader, bool) {
	if ctx == nil {
		return nil, false
	}
	loader, ok := ctx.Value(commandLoaderKey{}).(commandLoader)
	return loader, ok
}

func (c *Context) setInputEntity(entity CedarEntity) {
	if c == nil || entity == nil {
		return
	}
	c.Context = context.WithValue(c.Context, commandInputKey{}, entity)
}

func (c *Context) InputEntity() (CedarEntity, bool) {
	if c == nil {
		return nil, false
	}
	entity, ok := c.Context.Value(commandInputKey{}).(CedarEntity)
	return entity, ok
}

func (c *Context) setOutputEntity(entity CedarEntity) {
	if c == nil || entity == nil {
		return
	}
	c.Context = context.WithValue(c.Context, commandOutputKey{}, entity)
}

func (c *Context) OutputEntity() (CedarEntity, bool) {
	if c == nil {
		return nil, false
	}
	entity, ok := c.Context.Value(commandOutputKey{}).(CedarEntity)
	return entity, ok
}
