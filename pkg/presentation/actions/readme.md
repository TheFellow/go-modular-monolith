# Action presentation model

`actions` turns declarative control requirements into presentation state shared by GUI, TUI, and
web adapters. It deliberately does not execute commands: command middleware remains the security
boundary, and presentation evaluation only improves what the user sees.

A surface declares stable control IDs in `Group`s. The group's permission is inherited by nested
groups and controls; `Require` replaces that permission and `Public` explicitly removes it. This
makes broad rules concise while supporting exceptional actions such as publish:

```go
declaration := actions.Group{
	Permission: actions.Require(canEdit),
	Controls: []actions.Control{
		{ID: "name"},
		{
			ID:         "publish",
			Permission: actions.Require(canPublish),
			Conditions: []actions.Condition{saved, publishable},
		},
	},
}

states, err := actions.Evaluate(ctx, declaration)
```

The evaluator applies these rules consistently:

- a typed permission denial produces `Visible: false` and does not run conditions;
- an authorization evaluation failure is returned as an error rather than disguised as denial;
- an authorized but unmet condition produces a visible, disabled control and its first disabled
  reason; and
- empty or duplicate IDs and invalid declarations return errors early.

Adapters choose how hidden controls, disabled styling, tooltips, status text, or shortcut help are
rendered. Conditions belong in the domain surface adapter when they are view-specific; reusable
business predicates may live closer to the domain. In every case, the corresponding command must
independently enforce authorization and business invariants.

Do not authorize a collection control against a fabricated entity. When list authorization is
evaluated per returned entity (including ABAC-based row elision), entering the collection is a
public presentation capability and each returned row remains protected by the query pipeline. Use
an authorized collection control only when the policy model defines a genuine collection resource.

A projection is a snapshot for one principal and resource. Discard it when the actor or selection
changes, recompute it after persisted state changes, and clear it if evaluation fails so an earlier
enabled state cannot authorize a newer target. Permission runs before conditions intentionally:
denied controls do not evaluate or expose disabled reasons that could reveal domain prerequisites.

## Adding an action

1. Add the domain operation to its Cedar schema and policy, then regenerate authorization code.
2. Give the control a stable, namespaced `actions.ID` in the domain package.
3. Add it to the domain projector. Inherit the containing permission only when it is truly the same
   operation; otherwise use `Require` for the action's own Cedar check.
4. Add domain conditions that return a useful disabled reason. Keep dirty, submitting, confirming,
   focus, and similar runtime state in the concrete UI adapter.
5. Bind `Visible` and `Enabled` to the native control or key help, surface evaluator errors, and test
   permit, deny, unavailable, and failure cases.
6. Enforce the same authorization and invariants in the command. Projection can become stale
   between rendering and execution.

For debugging, inspect the state by stable ID and determine which layer changed it: `Visible=false`
means a typed permission denial; `Visible=true, Enabled=false` and `DisabledReason` identify the
first failed condition; no result plus an error means evaluation failed. Empty/duplicate IDs and
nil checks are declaration defects, not user denials. There is currently no HTTP capability
endpoint; a web adapter can consume the same projector in process or expose it through a transport
designed by that application.

Run `go test ./pkg/presentation/actions` while iterating.
