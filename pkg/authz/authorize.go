package authz

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	cedar "github.com/cedar-policy/cedar-go"
)

var ErrDenied = errors.New("authorization denied")

type DeniedError struct {
	Principal  cedar.EntityUID
	Action     cedar.EntityUID
	Resource   cedar.EntityUID
	Diagnostic cedar.Diagnostic
}

func (e DeniedError) Error() string {
	parts := []string{
		"authz denied",
		fmt.Sprintf("principal=%s::%q", e.Principal.Type, e.Principal.ID),
		fmt.Sprintf("action=%s::%q", e.Action.Type, e.Action.ID),
		fmt.Sprintf("resource=%s::%q", e.Resource.Type, e.Resource.ID),
	}
	if len(e.Diagnostic.Errors) > 0 {
		parts = append(parts, fmt.Sprintf("errors=%d", len(e.Diagnostic.Errors)))
	}
	return strings.Join(parts, " ")
}

func (e DeniedError) Unwrap() error { return ErrDenied }

var (
	policiesOnce sync.Once
	policiesSet  *cedar.PolicySet
	policiesErr  error
)

func getPolicySet() (*cedar.PolicySet, error) {
	policiesOnce.Do(func() {
		policiesSet = cedar.NewPolicySet()

		docs := policyDocuments()
		files := make([]string, 0, len(docs))
		for file := range docs {
			files = append(files, file)
		}
		sort.Strings(files)

		for _, file := range files {
			body := docs[file]
			ps, err := cedar.NewPolicySetFromBytes(file, []byte(body))
			if err != nil {
				policiesErr = err
				return
			}

			for id, p := range ps.All() {
				_ = policiesSet.Add(cedar.PolicyID(file+":"+string(id)), p)
			}
		}
	})
	return policiesSet, policiesErr
}

func Authorize(ctx context.Context, principal cedar.EntityUID, action cedar.EntityUID) error {
	_ = ctx

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
		return fmt.Errorf("authz evaluation error: %s", diagnostic.Errors[0].Message)
	}
	if decision == cedar.Deny {
		return DeniedError{
			Principal:  principal,
			Action:     action,
			Resource:   resource,
			Diagnostic: diagnostic,
		}
	}
	return nil
}

func AuthorizeWithEntity(ctx context.Context, principal cedar.EntityUID, action cedar.EntityUID, resource cedar.Entity) error {
	_ = ctx

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
		return fmt.Errorf("authz evaluation error: %s", diagnostic.Errors[0].Message)
	}
	if decision == cedar.Deny {
		return DeniedError{
			Principal:  principal,
			Action:     action,
			Resource:   resource.UID,
			Diagnostic: diagnostic,
		}
	}
	return nil
}
