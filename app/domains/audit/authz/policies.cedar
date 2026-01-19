// app/domains/audit/authz/policies.cedar

// Audit logs are owner-only.
forbid(
    principal == Mixology::Actor::"manager",
    action in [
        Mixology::AuditEntry::Action::"list",
        Mixology::AuditEntry::Action::"get"
    ],
    resource
);

forbid(
    principal == Mixology::Actor::"sommelier",
    action in [
        Mixology::AuditEntry::Action::"list",
        Mixology::AuditEntry::Action::"get"
    ],
    resource
);

forbid(
    principal == Mixology::Actor::"bartender",
    action in [
        Mixology::AuditEntry::Action::"list",
        Mixology::AuditEntry::Action::"get"
    ],
    resource
);

forbid(
    principal == Mixology::Actor::"anonymous",
    action in [
        Mixology::AuditEntry::Action::"list",
        Mixology::AuditEntry::Action::"get"
    ],
    resource
);
