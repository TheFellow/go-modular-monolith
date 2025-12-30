// app/drinks/authz/policies.cedar

// Read operations are open for now.
permit(
    principal,
    action == Mixology::Drinks::Action::"list",
    resource
);

permit(
    principal,
    action == Mixology::Drinks::Action::"get",
    resource
);

// Write operations require an owner principal.
permit(
    principal == Mixology::Actor::"owner",
    action == Mixology::Drinks::Action::"create",
    resource
);

