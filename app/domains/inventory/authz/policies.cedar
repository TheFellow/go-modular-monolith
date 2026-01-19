// Inventory (stock) policies

// Anyone can read inventory.
permit(
    principal is Mixology::Actor,
    action in [
        Mixology::Inventory::Action::"list",
        Mixology::Inventory::Action::"get"
    ],
    resource is Mixology::Inventory
);

// Managers can modify inventory.
permit(
    principal == Mixology::Actor::"manager",
    action in [
        Mixology::Inventory::Action::"adjust",
        Mixology::Inventory::Action::"set"
    ],
    resource is Mixology::Inventory
);
