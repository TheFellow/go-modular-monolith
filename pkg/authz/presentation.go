package authz

import (
	"context"

	cedar "github.com/cedar-policy/cedar-go"
)

// EntityAuthorizer evaluates whether principal may perform action on resource.
// Presentation capability projectors accept this function type so applications
// may use the in-process policy set or adapt a remote policy service.
type EntityAuthorizer func(context.Context, cedar.EntityUID, cedar.EntityUID, cedar.Entity) error

// AuthorizeEntity evaluates the application's Cedar policies. The context is
// accepted to give callers one reusable boundary for local and remote policy
// evaluators; the in-process evaluator does not currently need it.
func AuthorizeEntity(_ context.Context, principal, action cedar.EntityUID, resource cedar.Entity) error {
	return AuthorizeWithEntity(principal, action, resource)
}
