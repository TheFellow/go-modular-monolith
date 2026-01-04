package authz_test

import (
	"context"
	"errors"
	"testing"

	drinksauthz "github.com/TheFellow/go-modular-monolith/app/domains/drinks/authz"
	"github.com/TheFellow/go-modular-monolith/pkg/authn"
	"github.com/TheFellow/go-modular-monolith/pkg/authz"
	cedar "github.com/cedar-policy/cedar-go"
)

func TestAuthorize_AllowsAnonymousList(t *testing.T) {
	t.Parallel()

	err := authz.Authorize(context.Background(), authn.Anonymous(), drinksauthz.ActionList)
	if err != nil {
		t.Fatalf("expected allow, got %v", err)
	}
}

func TestAuthorizeWithEntity_DeniesAnonymousCreate(t *testing.T) {
	t.Parallel()

	resource := cedar.Entity{
		UID:        cedar.NewEntityUID(cedar.EntityType("Mixology::Drink::Catalog"), cedar.String("default")),
		Parents:    cedar.NewEntityUIDSet(),
		Attributes: cedar.NewRecord(nil),
		Tags:       cedar.NewRecord(nil),
	}

	err := authz.AuthorizeWithEntity(context.Background(), authn.Anonymous(), drinksauthz.ActionCreate, resource)
	if !errors.Is(err, authz.ErrDenied) {
		t.Fatalf("expected ErrDenied, got %v", err)
	}
}

func TestAuthorizeWithEntity_AllowsOwnerCreate(t *testing.T) {
	t.Parallel()

	resource := cedar.Entity{
		UID:        cedar.NewEntityUID(cedar.EntityType("Mixology::Drink::Catalog"), cedar.String("default")),
		Parents:    cedar.NewEntityUIDSet(),
		Attributes: cedar.NewRecord(nil),
		Tags:       cedar.NewRecord(nil),
	}

	err := authz.AuthorizeWithEntity(context.Background(), authn.Owner(), drinksauthz.ActionCreate, resource)
	if err != nil {
		t.Fatalf("expected allow, got %v", err)
	}
}
