// Inventory (stock) policies

permit(
    principal == Mixology::Actor::"owner",
    action in [
        Mixology::Inventory::Action::"adjust",
        Mixology::Inventory::Action::"set"
    ],
    resource is Mixology::Inventory
);

permit(
    principal is Mixology::Actor,
    action in [
        Mixology::Inventory::Action::"list",
        Mixology::Inventory::Action::"get"
    ],
    resource is Mixology::Inventory
);
