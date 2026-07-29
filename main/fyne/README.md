# Fyne Desktop Application

This directory composes Mixology's native desktop client. It owns process
lifecycle, the desktop data directory, the root window, and application-level
navigation. Domain behavior belongs in `app/domains/*/surfaces/fyne` and enters
the shell as a `pkg/fyne.View`.

Every application workspace is supplied by a domain-owned, tested vertical
slice rather than accumulating domain behavior here.

## Boundaries

- `pkg/fyne` contains reusable Fyne mechanics and imports neither `app` nor
  `main`.
- `main/fyne` creates the application session and composes routes.
- `app/domains/<domain>/surfaces/fyne` adapts that domain's public module API
  into framework-native state, actions, and widgets.
- Concrete CLI, TUI, and Fyne presentation packages do not depend on one another.

These directions are enforced by `.arch-lint.yaml`.

## Persistence and lifecycle

The desktop executable stores `mixology.db` and `mixology.log` beneath the
operating system's user configuration directory in a `Mixology` directory.
This is intentional application-state semantics: both files belong to one
portable per-user state root, rather than splitting the embedded database and
diagnostic log across platform-specific cache, config, and data locations.
The Fyne window owns the application session. Closing it releases the embedded
database and log before the process exits.

The default state locations are:

| Platform | Directory |
| --- | --- |
| macOS | `~/Library/Application Support/Mixology` |
| Windows | `%AppData%\Mixology` |
| Linux | `$XDG_CONFIG_HOME/Mixology`, or `~/.config/Mixology` when unset |

The desktop database is deliberately separate from the CLI/TUI database at
`data/mixology.db`. Seeded CLI data therefore does not appear in the desktop
client, and deleting one database does not reset the other surface. For a
fresh desktop environment, close Mixology and move or remove both
`mixology.db` and `mixology.log` from the directory above.

## Keyboard and accessibility

Mixology exposes native menu equivalents for its application commands. On
macOS, `Primary` means Command; on Windows and Linux it means Control.

| Action | Shortcut |
| --- | --- |
| Refresh the current workspace | Primary+R |
| Start a new item, where supported | Primary+N |
| Save or submit the active editor | Primary+S |
| Cancel or go back | Escape |
| Navigate Dashboard through Tags | Alt+1 through Alt+8 |
| Quit | Primary+Q |

Shortcuts use the same enabled controls as pointer input. They do nothing when
an action is hidden, disabled, submitting, confirming, or invalid for the
current workspace mode. Tab and Shift+Tab use Fyne's standard focus traversal.

Fyne does not currently provide complete cross-platform screen-reader
semantics for every widget. Labels, form items, visible button text, native
menus, keyboard traversal, status text, and disabled states improve usable
structure, but they are not a claim of WCAG or platform accessibility
conformance. In particular, automated headless tests cannot validate VoiceOver,
Narrator, Orca, high-contrast rendering, OS text scaling, or spoken status and
dialog announcements.

Before each desktop release, manually audit on every supported operating
system:

- Complete each workspace using only Tab, Shift+Tab, Enter/Space, Escape, and
  the documented shortcuts; ensure focus remains visible and never enters
  hidden or disabled controls.
- Verify initial editor focus, validation-error recovery, confirmation-dialog
  focus, and focus restoration after cancel, save, deletion, and navigation.
- Inspect every control name, form label, status/error announcement, table or
  list reading order, and dialog with VoiceOver (macOS), Narrator (Windows),
  and Orca (Linux).
- Check light/dark and high-contrast themes, 200% display scaling, keyboard
  layouts that do not place digits identically, reduced-motion preferences,
  and minimum window size.
- Confirm native menu placement and Primary+Q behavior follow platform
  convention, and record any Fyne/backend limitation in the release notes.

## Run from source

Fyne needs Go, a C compiler, and native graphics headers at development time.
On macOS install Xcode command-line tools (`xcode-select --install`). On
Windows use the MSYS2 MinGW 64-bit toolchain and ensure
`C:\msys64\mingw64\bin` is on `PATH`. Debian and Ubuntu users can install the
Linux requirements with:

```sh
sudo apt-get install gcc libgl1-mesa-dev libwayland-dev libxkbcommon-dev xorg-dev
```

From the repository root:

```sh
go run ./main/fyne
```

The desktop defaults to the `owner` persona. Select the same authorization
personas supported by the CLI and TUI with `-actor` (or its `-as` alias):

```sh
go run ./main/fyne -actor bartender
go run ./main/fyne -as anonymous
```

Run `go run ./main/fyne -help` for the complete startup options. The selected
persona is fixed for that process, so restart the desktop to exercise another
authorization policy.

The first native build is slower because it compiles Fyne's C bindings.

## Build and package

[`FyneApp.toml`](FyneApp.toml) is the release metadata source. Before a
release, update `Details.Version` using semantic versioning and increment the
positive integer `Details.Build`. Packaging uses an exact Fyne tool version so
a checkout is reproducible; upgrade that version in this document and CI in
the same change after reviewing its release notes.

Run the command for the current host operating system from this directory:

```sh
cd main/fyne

# macOS: creates Mixology.app
go build -o Mixology .
go run fyne.io/tools/cmd/fyne@v1.7.2 package -os darwin -release -exe Mixology

# Windows: creates Mixology.exe with application resources
go build -o Mixology.exe .
go run fyne.io/tools/cmd/fyne@v1.7.2 package -os windows -release -exe Mixology.exe

# Linux: creates Mixology.tar.xz
go build -o Mixology .
go run fyne.io/tools/cmd/fyne@v1.7.2 package -os linux -release -exe Mixology
```

Build each package on its target operating system. The Linux archive contains
the executable, icon, and generated freedesktop entry beneath `usr/local` (or
`usr` on systems without `/usr/local`). Its desktop entry is generated from
the `LinuxAndBSD` metadata; downstream distro packages may relocate those
files to their conventional prefix.

CI publishes these outputs as **unsigned** review artifacts. The macOS bundle
is archived before upload so its executable modes and bundle metadata survive
the artifact round trip. Public macOS
distribution still requires a Developer ID Application signature, hardened
runtime, notarization with `notarytool`, and stapling. Windows distribution
requires an Authenticode code-signing certificate and signing the final
executable (for example with `signtool`). Those identity-backed steps belong
in a protected release workflow once signing secrets and release policy
exist; local and pull-request builds must not pretend to be trusted releases.

## Tests

Desktop composition tests use Fyne's in-memory test application and virtual
window. They exercise real widgets, navigation, persistence startup, and clean
shutdown without opening a platform window or requiring a display server.
Asynchronous view models receive both `pkg/fyne.Executor` and
`pkg/fyne.Dispatcher`; production uses background goroutines followed by the
Fyne event loop. Tests use deterministic FIFO or out-of-order execution and
semantic controls from `pkg/testutil/fynetest` while interacting with real
widgets.

```sh
go test -tags ci -race \
  ./pkg/fyne ./pkg/testutil/fynetest ./main/fyne \
  ./app/domains/audit/surfaces/fyne \
  ./app/domains/drinks/surfaces/fyne \
  ./app/domains/ingredients/surfaces/fyne \
  ./app/domains/inventory/surfaces/fyne \
  ./app/domains/menus/surfaces/fyne \
  ./app/domains/orders/surfaces/fyne \
  ./app/domains/tagging/surfaces/fyne
go tool arch-lint -config=.arch-lint.yaml
```

The `ci` build tag selects Fyne's in-memory driver. It does not replace the
ordinary native build (`go build ./main/fyne`), which verifies the host's real
graphics integration.

## Troubleshooting

- `Package gl was not found`, missing X11 headers, or Wayland compiler errors
  mean the Linux development packages above are incomplete.
- `gcc` or `cgo: C compiler not found` on Windows means the MinGW `bin`
  directory is missing from `PATH`; restart the terminal after changing it.
- A macOS compiler or SDK error normally means Xcode command-line tools are
  absent or need accepting/updating.
- `timeout: failed to run command ...` or display-server errors during tests
  usually mean headless tests were run without `-tags ci`.
- `timeout` or an open failure for `mixology.db` usually means another
  Mixology process still owns the embedded database. Close that process before
  retrying.
