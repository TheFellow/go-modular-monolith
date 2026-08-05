# Action presentation toolkit

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

Run `go test ./pkg/toolkits/actions` while iterating.
