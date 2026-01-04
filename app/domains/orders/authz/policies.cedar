// app/domains/orders/authz/policies.cedar

permit(
    principal is Mixology::Actor,
    action in [
        Mixology::Order::Action::"list",
        Mixology::Order::Action::"get"
    ],
    resource is Mixology::Order
);

permit(
    principal == Mixology::Actor::"owner",
    action in [
        Mixology::Order::Action::"place",
        Mixology::Order::Action::"complete",
        Mixology::Order::Action::"cancel"
    ],
    resource is Mixology::Order
);

