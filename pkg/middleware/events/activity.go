package events

import (
	"slices"
	"time"

	cedar "github.com/cedar-policy/cedar-go"
)

type Activity struct {
	WorkflowID   string
	Effects      []Effect
	Participants []cedar.EntityUID
	Action       cedar.EntityUID
	Resource     cedar.EntityUID
	Principal    cedar.EntityUID

	StartedAt   time.Time
	CompletedAt time.Time

	Touches []cedar.EntityUID

	Success bool
	Error   string
}

func NewActivity(action, resource, principal cedar.EntityUID) *Activity {
	return &Activity{
		Action:    action,
		Resource:  resource,
		Principal: principal,
		StartedAt: time.Now(),
		Touches:   make([]cedar.EntityUID, 0, 8),
	}
}

func (a *Activity) Touch(uid cedar.EntityUID) {
	if a == nil {
		return
	}
	if slices.Contains(a.Touches, uid) {
		return
	}
	a.Touches = append(a.Touches, uid)
}

func (a *Activity) Complete(err error) {
	if a == nil {
		return
	}
	a.CompletedAt = time.Now()
	a.Success = err == nil
	if err != nil {
		a.Error = err.Error()
	}
}

// Effect is a domain-authored explanation, not a persistence snapshot. On a
// failed activity its values describe attempted effects, never committed state.
type Effect struct {
	Kind     string
	Resource cedar.EntityUID
	Changes  []Change
}
type Change struct {
	Field  string
	Before string
	After  string
}
