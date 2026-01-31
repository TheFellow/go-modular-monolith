package authz

import (
	"sort"
	"sync"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	cedar "github.com/cedar-policy/cedar-go"
)

var (
	policiesOnce sync.Once
	policiesSet  *cedar.PolicySet
	policiesErr  error
)

func getPolicySet() (*cedar.PolicySet, error) {
	policiesOnce.Do(func() {
		policiesSet = cedar.NewPolicySet()

		docs := policyDocuments()
		sort.Slice(docs, func(i, j int) bool { return docs[i].Name < docs[j].Name })

		for _, doc := range docs {
			ps, err := cedar.NewPolicySetFromBytes(doc.Name, []byte(doc.Text))
			if err != nil {
				policiesErr = err
				return
			}

			for id, p := range ps.All() {
				_ = policiesSet.Add(cedar.PolicyID(doc.Name+":"+string(id)), p)
			}
		}
	})
	return policiesSet, policiesErr
}

// Authorize evaluates authorization for the given principal and action.
// This is a pure function with no logging or telemetry side effects.
// Observability should be handled by middleware wrapping this call.
func Authorize(principal cedar.EntityUID, action cedar.EntityUID) error {
	ps, err := getPolicySet()
	if err != nil {
		return err
	}

	resource := cedar.NewEntityUID(cedar.EntityType("Mixology::AuthZ::Query"), cedar.String("unused"))

	entities := cedar.EntityMap{
		principal: {
			UID:        principal,
			Parents:    cedar.NewEntityUIDSet(),
			Attributes: cedar.NewRecord(nil),
			Tags:       cedar.NewRecord(nil),
		},
		resource: {
			UID:        resource,
			Parents:    cedar.NewEntityUIDSet(),
			Attributes: cedar.NewRecord(nil),
			Tags:       cedar.NewRecord(nil),
		},
	}

	req := cedar.Request{
		Principal: principal,
		Action:    action,
		Resource:  resource,
		Context:   cedar.NewRecord(nil),
	}

	decision, diagnostic := cedar.Authorize(ps, entities, req)
	if len(diagnostic.Errors) > 0 {
		return errors.Internalf("authz evaluation error: %s", diagnostic.Errors[0].Message)
	}
	if decision == cedar.Deny {
		return errors.Permissionf(
			"authz denied principal=%s::%q action=%s::%q resource=%s::%q",
			principal.Type, principal.ID,
			action.Type, action.ID,
			resource.Type, resource.ID,
		)
	}
	return nil
}

// AuthorizeWithEntity evaluates authorization for the given principal, action, and resource.
// This is a pure function with no logging or telemetry side effects.
// Observability should be handled by middleware wrapping this call.
func AuthorizeWithEntity(principal cedar.EntityUID, action cedar.EntityUID, resource cedar.Entity) error {
	ps, err := getPolicySet()
	if err != nil {
		return err
	}

	entities := cedar.EntityMap{
		principal: {
			UID:        principal,
			Parents:    cedar.NewEntityUIDSet(),
			Attributes: cedar.NewRecord(nil),
			Tags:       cedar.NewRecord(nil),
		},
		resource.UID: resource,
	}

	req := cedar.Request{
		Principal: principal,
		Action:    action,
		Resource:  resource.UID,
		Context:   cedar.NewRecord(nil),
	}

	decision, diagnostic := cedar.Authorize(ps, entities, req)
	if len(diagnostic.Errors) > 0 {
		return errors.Internalf("authz evaluation error: %s", diagnostic.Errors[0].Message)
	}
	if decision == cedar.Deny {
		return errors.Permissionf(
			"authz denied principal=%s::%q action=%s::%q resource=%s::%q",
			principal.Type, principal.ID,
			action.Type, action.ID,
			resource.UID.Type, resource.UID.ID,
		)
	}
	return nil
}
