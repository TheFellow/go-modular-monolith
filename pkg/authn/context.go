package authn

import (
	"context"

	cedar "github.com/cedar-policy/cedar-go"
)

type principalKey struct{}

func ToContext(ctx context.Context, principal cedar.EntityUID) context.Context {
	return context.WithValue(ctx, principalKey{}, principal)
}

func FromContext(ctx context.Context) cedar.EntityUID {
	principal, ok := ctx.Value(principalKey{}).(cedar.EntityUID)
	if !ok {
		panic("no principal in context")
	}
	return principal
}
