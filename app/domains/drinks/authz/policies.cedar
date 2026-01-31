// app/domains/drinks/authz/policies.cedar

// Anyone can read drinks.
permit(
    principal,
    action in [
        Mixology::Drink::Action::"list",
        Mixology::Drink::Action::"get"
    ],
    resource
);

// Managers can manage all drinks.
permit(
    principal == Mixology::Actor::"manager",
    action in [
        Mixology::Drink::Action::"create",
        Mixology::Drink::Action::"update",
        Mixology::Drink::Action::"delete"
    ],
    resource is Mixology::Drink
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

// Bartenders can manage non-wine drinks.
permit(
    principal == Mixology::Actor::"bartender",
    action in [
        Mixology::Drink::Action::"create",
        Mixology::Drink::Action::"update",
        Mixology::Drink::Action::"delete"
    ],
    resource is Mixology::Drink
) when {
    resource.Category != "wine"
};
