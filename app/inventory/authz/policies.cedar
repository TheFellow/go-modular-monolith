// Inventory (stock) policies

permit(
    principal == Mixology::Actor::"owner",
    action in [
        Mixology::Stock::Action::"adjust",
        Mixology::Stock::Action::"set"
    ],
    resource is Mixology::Stock
);

permit(
    principal is Mixology::Actor,
    action in [
        Mixology::Stock::Action::"list",
        Mixology::Stock::Action::"get"
    ],
    resource is Mixology::Stock
);

