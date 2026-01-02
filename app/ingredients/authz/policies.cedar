// app/ingredients/authz/policies.cedar

// Read operations are open for now.
permit(
    principal,
    action == Mixology::Ingredient::Action::"list",
    resource
);

permit(
    principal,
    action == Mixology::Ingredient::Action::"get",
    resource
);

// Write operations require an owner principal.
permit(
    principal == Mixology::Actor::"owner",
    action == Mixology::Ingredient::Action::"create",
    resource
);

permit(
    principal == Mixology::Actor::"owner",
    action == Mixology::Ingredient::Action::"update",
    resource
);
