package audit_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/audit"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestPermissions_Audit(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		canRead bool
	}{
		{name: "owner", canRead: true},
		{name: "manager", canRead: false},
		{name: "sommelier", canRead: false},
		{name: "bartender", canRead: false},
		{name: "anonymous", canRead: false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := testutil.NewFixture(t)
			var ctx *middleware.Context
			if tc.name == "owner" {
				ctx = f.OwnerContext()
			} else {
				ctx = f.ActorContext(tc.name)
			}

			_, err := f.Audit.List(ctx, audit.ListRequest{})
			if tc.canRead {
				testutil.PermissionTestPass(t, err)
			} else {
				testutil.PermissionTestFail(t, err)
			}
		})
	}
}
