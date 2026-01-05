package authz

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/log"
	"github.com/TheFellow/go-modular-monolith/pkg/telemetry"
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

func Authorize(ctx context.Context, principal cedar.EntityUID, action cedar.EntityUID) error {
	start := time.Now()

	ps, err := getPolicySet()
	if err != nil {
		recordAuthZ(ctx, action, start, err)
		logDecision(log.FromContext(ctx), false, time.Since(start), err)
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
		err := errors.Internalf("authz evaluation error: %s", diagnostic.Errors[0].Message)
		recordAuthZ(ctx, action, start, err)
		logDecision(log.FromContext(ctx), false, time.Since(start), err)
		return err
	}
	if decision == cedar.Deny {
		err := errors.Permissionf(
			"authz denied principal=%s::%q action=%s::%q resource=%s::%q",
			principal.Type, principal.ID,
			action.Type, action.ID,
			resource.Type, resource.ID,
		)
		recordAuthZ(ctx, action, start, err)
		logDecision(log.FromContext(ctx), false, time.Since(start), nil)
		return err
	}
	recordAuthZ(ctx, action, start, nil)
	logDecision(log.FromContext(ctx), true, time.Since(start), nil)
	return nil
}

func AuthorizeWithEntity(ctx context.Context, principal cedar.EntityUID, action cedar.EntityUID, resource cedar.Entity) error {
	start := time.Now()

	ps, err := getPolicySet()
	if err != nil {
		recordAuthZ(ctx, action, start, err)
		logDecision(log.FromContext(ctx), false, time.Since(start), err)
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
		err := errors.Internalf("authz evaluation error: %s", diagnostic.Errors[0].Message)
		recordAuthZ(ctx, action, start, err)
		logDecision(log.FromContext(ctx), false, time.Since(start), err)
		return err
	}
	if decision == cedar.Deny {
		err := errors.Permissionf(
			"authz denied principal=%s::%q action=%s::%q resource=%s::%q",
			principal.Type, principal.ID,
			action.Type, action.ID,
			resource.UID.Type, resource.UID.ID,
		)
		recordAuthZ(ctx, action, start, err)
		logDecision(log.FromContext(ctx), false, time.Since(start), nil)
		return err
	}
	recordAuthZ(ctx, action, start, nil)
	logDecision(log.FromContext(ctx), true, time.Since(start), nil)
	return nil
}

func recordAuthZ(ctx context.Context, action cedar.EntityUID, start time.Time, err error) {
	m := telemetry.FromContext(ctx)
	actionLabel := actionLabel(action)

	m.Histogram(telemetry.MetricAuthZLatency, telemetry.LabelAction).
		ObserveDuration(start, actionLabel)

	switch {
	case err == nil:
		m.Counter(telemetry.MetricAuthZTotal, telemetry.LabelAction, telemetry.LabelDecision).
			Inc(actionLabel, "allow")
	case errors.IsPermission(err):
		m.Counter(telemetry.MetricAuthZTotal, telemetry.LabelAction, telemetry.LabelDecision).
			Inc(actionLabel, "deny")
		m.Counter(telemetry.MetricAuthZDenied, telemetry.LabelAction).
			Inc(actionLabel)
	default:
		m.Counter(telemetry.MetricAuthZTotal, telemetry.LabelAction, telemetry.LabelDecision).
			Inc(actionLabel, "error")
	}
}

func actionLabel(action cedar.EntityUID) string {
	// Mixology::Drink::Action::"create" -> Drink.create
	s := action.String()
	parts := strings.Split(s, "::")
	if len(parts) < 4 {
		return s
	}
	domain := parts[1]
	id := strings.Trim(parts[len(parts)-1], `"`)
	return domain + "." + id
}
