// Tag discovery is a separate authorization vertical. Granting either action
// intentionally discloses matching entity types and IDs without consulting
// the owning domains' read policies.
permit(
    principal == Mixology::Actor::"owner",
    action in [
        Mixology::TagDiscovery::Action::"show",
        Mixology::TagDiscovery::Action::"summary"
    ],
    resource is Mixology::TagDiscovery
);
