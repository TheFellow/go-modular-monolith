// app/domains/orders/authz/policies.cedar

// Staff can read orders.
permit(
    principal == Mixology::Actor::"manager",
    action in [
        Mixology::Order::Action::"list",
        Mixology::Order::Action::"get"
    ],
    resource is Mixology::Order
);

permit(
    principal == Mixology::Actor::"sommelier",
    action in [
        Mixology::Order::Action::"list",
        Mixology::Order::Action::"get"
    ],
    resource is Mixology::Order
);

permit(
    principal == Mixology::Actor::"bartender",
    action in [
        Mixology::Order::Action::"list",
        Mixology::Order::Action::"get"
    ],
    resource is Mixology::Order
);

// Bartenders and managers can manage orders.
permit(
    principal == Mixology::Actor::"manager",
    action in [
        Mixology::Order::Action::"place",
        Mixology::Order::Action::"complete",
        Mixology::Order::Action::"cancel"
    ],
    resource is Mixology::Order
);

permit(
    principal == Mixology::Actor::"bartender",
    action in [
        Mixology::Order::Action::"place",
        Mixology::Order::Action::"complete",
        Mixology::Order::Action::"cancel"
    ],
    resource is Mixology::Order
);
