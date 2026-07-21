package authz_test

import (
	"testing"

	drinksauthz "github.com/TheFellow/go-modular-monolith/app/domains/drinks/authz"
	"github.com/TheFellow/go-modular-monolith/pkg/authn"
	"github.com/TheFellow/go-modular-monolith/pkg/authz"
	errorspkg "github.com/TheFellow/go-modular-monolith/pkg/errors"
	cedar "github.com/cedar-policy/cedar-go"
)

func TestAuthorizeWithEntity_AllowsAnonymousList(t *testing.T) {
	t.Parallel()

	resource := cedar.Entity{
		UID: cedar.NewEntityUID(cedar.EntityType("Mixology::Drink"), cedar.String("wine")),
		Attributes: cedar.NewRecord(cedar.RecordMap{
			"Category": cedar.String("wine"),
		}),
	}
	err := authz.AuthorizeWithEntity(authn.Anonymous(), drinksauthz.ActionList, resource)
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

	err := authz.AuthorizeWithEntity(authn.Anonymous(), drinksauthz.ActionCreate, resource)
	if !errorspkg.IsPermission(err) {
		t.Fatalf("expected IsPermission, got %v", err)
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

	err := authz.AuthorizeWithEntity(authn.Owner(), drinksauthz.ActionCreate, resource)
	if err != nil {
		t.Fatalf("expected allow, got %v", err)
	}
}
