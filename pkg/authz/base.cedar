// pkg/authz/base.cedar

// Owner can do anything.
permit(
    principal == Mixology::Actor::"owner",
    action,
    resource
);

// Allow anonymous users to login (placeholder; real authn later).
permit(
    principal == Mixology::Actor::"anonymous",
    action == Mixology::Action::"login",
    resource == Mixology::Auth::Session::"current"
);
