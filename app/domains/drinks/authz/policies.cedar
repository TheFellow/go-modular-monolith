// app/domains/drinks/authz/policies.cedar

// Read operations are open for now.
permit(
    principal,
    action == Mixology::Drink::Action::"list",
    resource
);

permit(
    principal,
    action == Mixology::Drink::Action::"get",
    resource
);

// Write operations require an owner principal.
permit(
    principal == Mixology::Actor::"owner",
    action == Mixology::Drink::Action::"create",
    resource
);

permit(
    principal == Mixology::Actor::"owner",
    action == Mixology::Drink::Action::"update",
    resource
);
