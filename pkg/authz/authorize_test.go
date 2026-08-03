package authz_test

import (
	"testing"

	drinksauthz "github.com/TheFellow/go-modular-monolith/app/domains/drinks/authz"
	"github.com/TheFellow/go-modular-monolith/pkg/authn"
	"github.com/TheFellow/go-modular-monolith/pkg/authz"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	cedar "github.com/cedar-policy/cedar-go"
)

func TestAuthorizeWithEntity_AllowsAnonymousList(t *testing.T) {
	t.Parallel()

	resource := validDrinkEntity()
	err := authz.AuthorizeWithEntity(authn.Anonymous(), drinksauthz.ActionList, resource)
	testutil.Ok(t, err)
}

func TestAuthorizeWithEntity_DeniesAnonymousCreate(t *testing.T) {
	t.Parallel()

	resource := validDrinkEntity()

	err := authz.AuthorizeWithEntity(authn.Anonymous(), drinksauthz.ActionCreate, resource)
	testutil.ErrorIsPermission(t, err)
}

func TestAuthorizeWithEntity_AllowsOwnerCreate(t *testing.T) {
	t.Parallel()

	resource := validDrinkEntity()

	err := authz.AuthorizeWithEntity(authn.Owner(), drinksauthz.ActionCreate, resource)
	testutil.Ok(t, err)
}

func TestAuthorizeWithEntity_RejectsInvalidResource(t *testing.T) {
	t.Parallel()

	resource := validDrinkEntity()
	resource.Attributes = cedar.NewRecord(cedar.RecordMap{
		drinksauthz.DrinkCategoryAttr: cedar.Long(42),
	})

	err := authz.AuthorizeWithEntity(authn.Owner(), drinksauthz.ActionGet, resource)
	testutil.ErrorIsInternal(t, err)
}

func TestAuthorizeWithEntity_RejectsUnknownResourceType(t *testing.T) {
	t.Parallel()

	err := authz.AuthorizeWithEntity(authn.Owner(), drinksauthz.ActionGet, cedar.Entity{
		UID: cedar.NewEntityUID("Unknown::Resource", "test"),
	})
	testutil.ErrorIsInternal(t, err)
}

func validDrinkEntity() cedar.Entity {
	return drinksauthz.Drink{
		UID:         cedar.NewEntityUID(drinksauthz.DrinkType, "wine"),
		Name:        "Wine",
		Category:    "wine",
		Glass:       "wine glass",
		Description: "A test wine",
	}.CedarEntity()
}
