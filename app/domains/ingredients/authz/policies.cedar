// app/domains/ingredients/authz/policies.cedar

// Anyone can read ingredients.
permit(
    principal,
    action in [
        Mixology::Ingredient::Action::"list",
        Mixology::Ingredient::Action::"get"
    ],
    resource
);

// Managers can modify ingredients.
permit(
    principal == Mixology::Actor::"manager",
    action in [
        Mixology::Ingredient::Action::"create",
        Mixology::Ingredient::Action::"update",
        Mixology::Ingredient::Action::"delete"
    ],
    resource is Mixology::Ingredient
);
