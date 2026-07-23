package testutil

import (
	"slices"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/audit"
	"github.com/TheFellow/go-modular-monolith/app/domains/audit/models"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func (f *Fixture) LatestAuditEntry(action cedar.EntityUID) *models.AuditEntry {
	f.T.Helper()

	page, err := f.Audit.List(f.OwnerContext(), audit.ListRequest{Action: action})
	Ok(f.T, err)
	NotEquals(f.T, len(page.Items), 0)
	return slices.MaxFunc(page.Items, func(a, b *models.AuditEntry) int {
		return a.StartedAt.Compare(b.StartedAt)
	})
}

func AuditTouches(t testing.TB, entry *models.AuditEntry, want ...cedar.EntityUID) {
	t.Helper()
	Equals(t, entry.Touches, want, cmpopts.SortSlices(func(a, b cedar.EntityUID) bool {
		return a.String() < b.String()
	}))
}
