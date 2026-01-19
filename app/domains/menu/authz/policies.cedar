// app/domains/menu/authz/policies.cedar

permit(
    principal,
    action in [
        Mixology::Menu::Action::"list",
        Mixology::Menu::Action::"get"
    ],
    resource
);

// Managers can manage menus.
permit(
    principal == Mixology::Actor::"manager",
    action in [
        Mixology::Menu::Action::"create",
        Mixology::Menu::Action::"add_drink",
        Mixology::Menu::Action::"remove_drink",
        Mixology::Menu::Action::"publish"
    ],
    resource is Mixology::Menu
);
