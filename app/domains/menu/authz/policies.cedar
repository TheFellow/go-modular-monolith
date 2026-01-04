// app/domains/menu/authz/policies.cedar

permit(
    principal,
    action == Mixology::Menu::Action::"list",
    resource
);

permit(
    principal,
    action == Mixology::Menu::Action::"get",
    resource
);

permit(
    principal == Mixology::Actor::"owner",
    action == Mixology::Menu::Action::"create",
    resource
);

permit(
    principal == Mixology::Actor::"owner",
    action == Mixology::Menu::Action::"add_drink",
    resource
);

permit(
    principal == Mixology::Actor::"owner",
    action == Mixology::Menu::Action::"remove_drink",
    resource
);

permit(
    principal == Mixology::Actor::"owner",
    action == Mixology::Menu::Action::"publish",
    resource
);
