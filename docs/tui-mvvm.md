# TUI MVVM conventions

The TUI uses standard presentation contracts for recurring interface shapes and
keeps domain behavior in domain ViewModels. This borrows the useful boundary
from CODE Framework's standard views without introducing inheritance or generic
business models.

## Boundaries

- Domain models and application commands remain concrete and domain-owned.
- `pkg/tui/mvvm` contains presentation-only values and interaction contracts.
- Reusable components own recurring terminal mechanics such as list rendering,
  forms, dialogs, sizing, and input ownership.
- Domain ViewModels adapt domain values, select commands, and handle domain
  results. They do not invent alternate versions of shared contracts.

Standard presentation fields are semantic (`Title`, `Description`, and
`FilterValue`), rather than numbered generic fields. A standard contract should
only be introduced after multiple screens demonstrate the same behavior.

## Interaction contract

Every root ViewModel reports its current `Interaction` explicitly. The root
application uses it to decide whether printable keys and Back belong to the
active view. Help remains part of the ViewModel contract because it describes
the actions available in that same state.

Tests should assert the interaction contract together with the observable key
behavior. A mode that captures text must not allow printable global shortcuts;
a mode that owns Back must consume it before application navigation.

## Testing standard views

Shared presentation types have focused contract tests. Each adopting domain
retains tests for its mapping and commands. Root ViewModels additionally run a
common conformance suite covering passive browsing, text capture, Back
ownership, and help validity. True Bubble Tea driver fixtures remain responsible
for end-to-end command scheduling, rendering, and navigation.

The intent is regularity, not uniformity. Specialized workspaces may render
differently, but they use the same interaction vocabulary and must satisfy the
same conformance checks.
