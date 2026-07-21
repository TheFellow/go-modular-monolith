// app/domains/drinks/authz/policies.cedar

// Managers can read all drinks.
permit(
    principal == Mixology::Actor::"manager",
    action in [
        Mixology::Drink::Action::"list",
        Mixology::Drink::Action::"get"
    ],
    resource is Mixology::Drink
);

// Anonymous users can browse the full public drink catalog.
permit(
    principal == Mixology::Actor::"anonymous",
    action in [
        Mixology::Drink::Action::"list",
        Mixology::Drink::Action::"get"
    ],
    resource is Mixology::Drink
);

// Sommeliers can read wine drinks.
permit(
    principal == Mixology::Actor::"sommelier",
    action in [
        Mixology::Drink::Action::"list",
        Mixology::Drink::Action::"get"
    ],
    resource is Mixology::Drink
) when {
    resource.Category == "wine"
};

// Bartenders can read non-wine drinks.
permit(
    principal == Mixology::Actor::"bartender",
    action in [
        Mixology::Drink::Action::"list",
        Mixology::Drink::Action::"get"
    ],
    resource is Mixology::Drink
) when {
    resource.Category != "wine"
};

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
