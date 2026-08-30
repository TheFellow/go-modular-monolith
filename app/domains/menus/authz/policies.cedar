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
        Mixology::Menu::Action::"update",
        Mixology::Menu::Action::"delete",
        Mixology::Menu::Action::"drink:add",
        Mixology::Menu::Action::"drink:remove",
        Mixology::Menu::Action::"publish",
        Mixology::Menu::Action::"draft",
        Mixology::Menu::Action::"readiness",
        Mixology::Menu::Action::"tag",
        Mixology::Menu::Action::"untag"
    ],
    resource is Mixology::Menu
);
