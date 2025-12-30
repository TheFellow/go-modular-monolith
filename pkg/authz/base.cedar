// pkg/authz/base.cedar

// Allow anonymous users to login (placeholder; real authn later).
permit(
    principal == Mixology::Actor::"anonymous",
    action == Mixology::Action::"login",
    resource == Mixology::Auth::Session::"current"
);

