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

// Sommeliers can manage wine drinks.
permit(
    principal == Mixology::Actor::"sommelier",
    action in [
        Mixology::Drink::Action::"create",
        Mixology::Drink::Action::"update",
        Mixology::Drink::Action::"delete"
    ],
    resource is Mixology::Drink
) when {
    resource.Category == "wine"
};

permit(
    principal == Mixology::Actor::"owner",
    action == Mixology::Drink::Action::"update",
    resource
);

// Bartenders can update cocktails.
permit(
    principal == Mixology::Actor::"bartender",
    action == Mixology::Drink::Action::"update",
    resource is Mixology::Drink
) when {
    resource.Category == "cocktail"
};

permit(
    principal == Mixology::Actor::"owner",
    action == Mixology::Drink::Action::"delete",
    resource
);
