// Package actions evaluates framework-neutral control presentation state.
//
// A UI surface declares groups of controls, assigns authorization checks at
// the broadest useful scope, and overrides them only where a control has a
// distinct permission. Authorization decides visibility; domain and transient
// conditions decide whether an authorized control is enabled.
package actions

import (
	"context"
	"fmt"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/set"
)

// ID is the stable identity of a control within a declaration. It should not
// be derived from a translated label or a control's position in a layout.
type ID string

// Authorize performs one permission check. A permission error means denied;
// any other error means authorization could not be evaluated.
type Authorize func(context.Context) error

// Condition evaluates a business or transient prerequisite after permission
// has been granted. A false result keeps the control visible but disabled.
// reason should tell the user what must change before the action is available.
type Condition func(context.Context) (available bool, reason string, err error)

type permissionMode uint8

const (
	permissionInherit permissionMode = iota
	permissionPublic
	permissionRequired
)

// Permission declares how a group or control obtains its authorization rule.
// Its zero value inherits the containing group's rule.
type Permission struct {
	mode      permissionMode
	authorize Authorize
}

// Inherit uses the containing group's permission. It is also the zero value.
func Inherit() Permission { return Permission{} }

// Public explicitly removes an inherited permission requirement.
func Public() Permission { return Permission{mode: permissionPublic} }

// Require replaces an inherited permission with authorize.
func Require(authorize Authorize) Permission {
	return Permission{mode: permissionRequired, authorize: authorize}
}

// Control describes a renderable action. Conditions are evaluated in order;
// evaluation stops at the first unavailable condition or error.
type Control struct {
	ID         ID
	Permission Permission
	Conditions []Condition
}

// Group applies a permission default to all directly and transitively
// contained controls. A nested group may inherit, replace, or remove it.
// Groups organize declarations only; they are not themselves rendered.
type Group struct {
	Permission Permission
	Controls   []Control
	Groups     []Group
}

// State is the presentation state a GUI, TUI, or web adapter consumes.
type State struct {
	ID             ID     `json:"id"`
	Visible        bool   `json:"visible"`
	Enabled        bool   `json:"enabled"`
	DisabledReason string `json:"disabled_reason,omitempty"`
}

// Evaluate resolves a declaration in depth-first declaration order. IDs must
// be non-empty and unique across the complete tree.
func Evaluate(ctx context.Context, declaration Group) ([]State, error) {
	states := make([]State, 0)
	var seen set.Set[ID]
	if err := evaluateGroup(ctx, declaration, nil, &seen, &states); err != nil {
		return nil, err
	}
	return states, nil
}

func evaluateGroup(ctx context.Context, group Group, inherited Authorize, seen *set.Set[ID], states *[]State) error {
	authorize, err := resolvePermission(group.Permission, inherited)
	if err != nil {
		return err
	}

	for _, control := range group.Controls {
		if control.ID == "" {
			return fmt.Errorf("actions: control ID must not be empty")
		}
		if seen.Contains(control.ID) {
			return fmt.Errorf("actions: duplicate control ID %q", control.ID)
		}
		seen.Add(control.ID)

		controlAuthorize, err := resolvePermission(control.Permission, authorize)
		if err != nil {
			return fmt.Errorf("actions: control %q: %w", control.ID, err)
		}
		state, err := evaluateControl(ctx, control, controlAuthorize)
		if err != nil {
			return fmt.Errorf("actions: control %q: %w", control.ID, err)
		}
		*states = append(*states, state)
	}
	for _, child := range group.Groups {
		if err := evaluateGroup(ctx, child, authorize, seen, states); err != nil {
			return err
		}
	}
	return nil
}

func resolvePermission(permission Permission, inherited Authorize) (Authorize, error) {
	switch permission.mode {
	case permissionInherit:
		return inherited, nil
	case permissionPublic:
		return nil, nil
	case permissionRequired:
		if permission.authorize == nil {
			return nil, fmt.Errorf("required permission has no authorization check")
		}
		return permission.authorize, nil
	default:
		return nil, fmt.Errorf("invalid permission mode %d", permission.mode)
	}
}

func evaluateControl(ctx context.Context, control Control, authorize Authorize) (State, error) {
	state := State{ID: control.ID, Visible: true, Enabled: true}
	if authorize != nil {
		if err := authorize(ctx); err != nil {
			if errors.IsPermission(err) {
				state.Visible = false
				state.Enabled = false
				return state, nil
			}
			return State{}, err
		}
	}

	for i, condition := range control.Conditions {
		if condition == nil {
			return State{}, fmt.Errorf("condition %d is nil", i)
		}
		available, reason, err := condition(ctx)
		if err != nil {
			return State{}, fmt.Errorf("condition %d: %w", i, err)
		}
		if !available {
			state.Enabled = false
			state.DisabledReason = reason
			return state, nil
		}
	}
	return state, nil
}
