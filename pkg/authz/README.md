# Cedar authorization

`pkg/authz` assembles Mixology's Cedar policy documents and evaluates authorization requests. Each
bounded context owns its schema, policies, and Cedar-facing resource model; this package supplies
the shared evaluator and generation workflow. The [architecture guide](../../docs/architecture.md#authorization)
describes where authorization sits in the command and query pipelines.

## How the pieces fit

```text
app/domains/<domain>/authz/schema.cedarschema + policies.cedar
        |
        | go generate ./...
        v
generated domain actions, resource model, validator, and tests
        |
        v
domain model CedarEntity() -> middleware -> AuthorizeWithEntity
        |
        v
permit, typed permission error, or typed internal error
```

The package-level [`base.cedar`](base.cedar) contains application-wide rules, including a permit for
the owner across actions and resources. Domain rules stay beside their owning context. At generation
time references to all embedded policy documents are assembled in
[`policies_gen.go`](policies_gen.go); at runtime they are parsed once into one policy set in
deterministic document order.

## Domain authorization contract

Every `app/domains/<domain>/authz` directory contains:

- `schema.cedarschema`, which declares the resource shape and actions;
- `policies.cedar`, which grants or forbids access;
- `policies.go`, which embeds that policy document; and
- generated `models_gen.go` and `models_gen_test.go` files.

Generation turns the schema into a small boundary API: resource and action type constants,
`Action...` entity UIDs, a Cedar-facing resource struct, `CedarEntity`, and `ValidateEntity`. The
domain model explicitly converts its business state to that generated representation. See the
[drink schema](../../app/domains/drinks/authz/schema.cedarschema),
[drink policies](../../app/domains/drinks/authz/policies.cedar), and
[`models.Drink.CedarEntity`](../../app/domains/drinks/models/drink.go) for a complete example.

That conversion is part of the security boundary. A field used by policy must be populated from
the state being authorized, and taggable resources must pass their string tags through. Tags have
no built-in authorization meaning; a policy gives a particular tag meaning, as described in the
[feature guide](../../docs/features.md#tags).

## Evaluation API

Most application code should use the middleware helpers rather than call the evaluator directly:

- `middleware.RunEntityQuery` authorizes a loaded entity before returning it.
- `middleware.RunPageQuery` authorizes each row, omits permission-denied rows, and propagates
  evaluation or storage failures.
- `middleware.RunCommand` authorizes the loaded and resulting resource states so policies can
  constrain transitions.

Those behaviors live in [`pkg/middleware`](../middleware/run.go). Presentation code may call
`AuthorizeWithEntity` to decide whether to expose an action, but the command/query pipeline remains
the enforcement point.

The lower-level call looks like this:

```go
resource := drinkauthz.Drink{
	UID:         cedar.NewEntityUID(drinkauthz.DrinkType, "drk-example"),
	Name:        "House Red",
	Category:    "wine",
	Glass:       "wine glass",
	Description: "",
}.CedarEntity()

err := authz.AuthorizeWithEntity(
	authn.Sommelier(),
	drinkauthz.ActionUpdate,
	resource,
)
```

`AuthorizeWithEntity` validates the resource against its registered domain schema before asking
Cedar for a decision. An unknown resource type, invalid resource shape, or Cedar diagnostic becomes
a typed internal error; a deny becomes a typed permission error. Generation normally catches
invalid policy syntax, while any failure to parse the embedded policy set at runtime propagates as
a setup error. Cedar is deny-by-default, so no matching permit is a denial. The evaluator itself
performs no logging, metrics, or audit work; middleware owns those side effects. See the
[application errors guide](../errors/README.md) for classification and safe presentation behavior.

`Authorize` is the action-only variant. It evaluates against the synthetic
`Mixology::AuthZ::Query::"unused"` resource and is suitable only for policies that do not require a
domain resource. Domain operations normally require `AuthorizeWithEntity`.

## Adding or changing a domain policy

1. Add or edit the domain's `schema.cedarschema`, `policies.cedar`, and embedding `policies.go`.
2. Run `go generate ./...` from the repository root. Do not edit generated models, tests, or the
   assembled policy registry by hand.
3. Implement or update the business model's `CedarEntity` conversion.
4. Route the generated action through the appropriate middleware helper.
5. Test representative permits, denials, resource attributes, tags, and state transitions.

The generator deliberately supports a narrow schema profile: one action namespace per domain, one
shared resource type for its actions, empty action contexts, and no resource parent types. Resource
attributes may be Cedar `String`, `Long`, `Bool`, supported scalar aliases, or entity references;
attributes cannot be optional. Tags, when present, must be strings. Generation fails on unsupported
shapes, invalid policies, or Go-name collisions rather than producing a partial boundary model.

Run the focused suites while iterating:

```sh
go generate ./...
go test ./pkg/authz/... ./app/domains/<domain>/authz
```

Commit schema and policy sources together with all generated changes. See the
[development guide](../../docs/development.md#everyday-loop) for the repository-wide generation
rule.
