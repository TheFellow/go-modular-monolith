# Sprint 005: Polish & Production Readiness

## Goal

Refine the TUI for production use: responsive layouts, mouse support, theming, comprehensive error handling, performance
optimization, accessibility considerations, and testing infrastructure.

## Problem

After Sprint 004, core functionality works but the TUI needs polish for real-world use. Edge cases aren't handled,
performance with large datasets is untested, and the UI doesn't adapt well to different terminal sizes.

## Solution

Address fit-and-finish issues, add optional mouse support, implement theming, optimize performance, and establish
testing patterns for TUI components.

## Tasks

### Phase 1: Responsive Layout

#### Terminal Size Handling

- [ ] Define minimum terminal size (80×24)
- [ ] Show clear message when terminal too small:
    ```
    Terminal too small. Please resize to at least 80×24.
    Current: 60×20
    ```
- [ ] Define breakpoints:
    - Compact: 80-99 columns (hide detail panes)
    - Standard: 100-139 columns (60/40 split)
    - Wide: 140+ columns (50/50 split with extra info)

#### Adaptive Layouts

- [ ] List views: Hide detail pane on narrow terminals
- [ ] Menu CreateViewModel: Stack panes vertically on narrow terminals
- [ ] Order CreateViewModel: Same vertical stacking
- [ ] Forms: Single column always (no horizontal split)
- [ ] Dashboard: Reduce card size, stack vertically if needed

#### Dynamic Resizing

- [ ] Handle `tea.WindowSizeMsg` in all views
- [ ] Smooth resize without flickering
- [ ] Preserve scroll position on resize
- [ ] Preserve selection on resize

### Phase 2: Mouse Support

#### Clickable Elements

- [ ] List items: Click to select
- [ ] Buttons: Click to activate
- [ ] Tabs/panes: Click to focus
- [ ] Form fields: Click to focus
- [ ] Links: Click to navigate (entity references)

#### Scroll Support

- [ ] Mouse wheel scrolls lists/viewports
- [ ] Scroll indicators show position
- [ ] Click-and-drag scrollbar (stretch goal)

#### Hover States

- [ ] Highlight item under cursor
- [ ] Show tooltips for truncated text (stretch goal)
- [ ] Different cursor styles for clickable vs. non-clickable (if terminal supports)

#### Mouse Configuration

- [ ] Enable mouse by default
- [ ] `--no-mouse` flag to disable
- [ ] Or detect terminal capabilities automatically

### Phase 3: Theming

#### Built-in Themes

- [ ] Create `Dark` theme (default):
    - Background: terminal default
    - Primary: cyan
    - Secondary: magenta
    - Success: green
    - Warning: yellow
    - Error: red
    - Muted: gray
- [ ] Create `Light` theme:
    - Adjusted colors for light backgrounds
    - Higher contrast where needed
- [ ] Create `Monochrome` theme:
    - No colors, uses bold/underline/reverse
    - For accessibility or preference

#### Theme Selection

- [ ] Auto-detect terminal background (if possible)
- [ ] `--theme <name>` flag to override
- [ ] Runtime theme switching (future: in-app settings)

#### Custom Theme Support (Stretch)

- [ ] Theme file format (YAML/JSON)
- [ ] Load custom themes from `~/.config/mixology/themes/`

### Phase 4: Error Handling & Recovery

#### Error Display

- [ ] Create dedicated `ErrorView` component:
    - Error message prominently displayed
    - Technical details collapsible
    - Retry button if applicable
    - Back button to return to previous view
- [ ] Status bar errors: Brief message + "[Press ? for details]"
- [ ] Distinguish error types:
    - Validation errors (inline, yellow)
    - Operation failures (status bar, red)
    - Connection errors (modal, retry option)
    - Fatal errors (full screen, exit option)

#### Graceful Degradation

- [ ] Handle app layer unavailability (connection to store fails)
- [ ] Show cached/stale data with warning if refresh fails
- [ ] Partial operation handling (some items failed in batch)

#### Recovery Actions

- [ ] Retry last failed operation (`r` in error state)
- [ ] Clear error and refresh (`R` force refresh)
- [ ] Return to last known good state

### Phase 5: Performance Optimization

#### Lazy Loading

- [ ] Don't load view data until view is accessed
- [ ] Paginate large lists (>100 items):
    - Load first page immediately
    - Load more as user scrolls (infinite scroll)
    - Show loading indicator at bottom
- [ ] Background refresh doesn't block UI

#### Rendering Optimization

- [ ] Cache rendered strings when data unchanged
- [ ] Diff-based updates (only re-render changed portions)
- [ ] Debounce rapid updates (e.g., during resize)

#### Memory Management

- [ ] Limit cached views (LRU eviction)
- [ ] Clear detail pane cache on navigation
- [ ] Release large data structures when views hidden

#### Benchmarks

- [ ] Test with 1000+ drinks
- [ ] Test with 100+ menus
- [ ] Test with 10000+ audit entries
- [ ] Profile and identify bottlenecks

### Phase 6: Accessibility

#### Screen Reader Considerations

- [ ] Ensure text-based output is meaningful
- [ ] Avoid relying solely on color for information
- [ ] Status badges include text (not just icons)
- [ ] Error messages are clear and actionable

#### Keyboard Navigation

- [ ] Tab order is logical
- [ ] Focus indicators are visible
- [ ] All actions have keyboard shortcuts
- [ ] No keyboard traps (can always escape/quit)

#### High Contrast

- [ ] Monochrome theme works without color
- [ ] Important text uses sufficient contrast
- [ ] Selected items distinguishable without color

### Phase 7: User Preferences

#### Session Persistence (Stretch)

- [ ] Remember last view on exit, restore on launch
- [ ] Remember filter settings per view
- [ ] Remember window pane sizes

#### Configuration File

- [ ] Support `~/.config/mixology/tui.yaml`:
    ```yaml
    theme: dark
    mouse: true
    confirmDeletes: true
    showStockWarnings: true
    refreshInterval: 0  # 0 = manual only
    ```
- [ ] Command-line flags override config file

### Phase 8: Testing Infrastructure

#### Unit Tests

- [ ] Test model state transitions
- [ ] Test message handling
- [ ] Test form validation
- [ ] Test navigation logic

#### Golden File Tests

- [ ] Capture View() output as golden files
- [ ] Compare rendered output in tests
- [ ] Update goldens with flag (`-update`)

#### Integration Tests

- [ ] Test full workflows with mock app layer
- [ ] Test keyboard sequences produce expected results
- [ ] Test error handling paths

#### Test Utilities

- [ ] Create `testutil.SendKeys(model, "abc\n")` helper
- [ ] Create `testutil.AssertViewContains(model, "text")` helper
- [ ] Create mock `app.Application` for testing

### Phase 9: Documentation

#### In-App Help

- [ ] `?` shows complete keybinding reference
- [ ] Context-sensitive help (different per view)
- [ ] First-run welcome/tutorial (optional)

#### README Updates

- [ ] Document `--tui` flag
- [ ] Screenshots of main views
- [ ] Keyboard shortcuts reference
- [ ] Configuration options

### Phase 10: Final Polish

#### Visual Refinements

- [ ] Consistent spacing throughout
- [ ] Aligned columns in lists
- [ ] Proper text truncation with ellipsis
- [ ] Loading states don't cause layout shift

#### Animation (Subtle)

- [ ] Spinner for loading states
- [ ] Brief flash on successful action
- [ ] Smooth cursor movement (if terminal supports)

#### Edge Cases

- [ ] Handle very long entity names (truncation)
- [ ] Handle empty states gracefully in all views
- [ ] Handle rapid key presses without lag
- [ ] Handle terminal disconnect/reconnect

## Acceptance Criteria

### Responsive Layout

- [ ] TUI works at 80×24 (minimum)
- [ ] TUI adapts gracefully to larger sizes
- [ ] Resize events handled without crash or corruption
- [ ] Clear message shown when terminal too small

### Mouse Support

- [ ] Click to select works in all lists
- [ ] Mouse wheel scrolling works
- [ ] `--no-mouse` disables mouse handling

### Theming

- [ ] Dark theme is default and looks good
- [ ] Light theme available via `--theme light`
- [ ] Monochrome theme works without color

### Error Handling

- [ ] All errors display meaningfully
- [ ] User can recover from errors
- [ ] App doesn't crash on API failures

### Performance

- [ ] 1000 items load in <1 second
- [ ] Scrolling is smooth (60fps feel)
- [ ] Memory usage stable during extended sessions

### Accessibility

- [ ] Full keyboard navigation possible
- [ ] Information conveyed without relying solely on color
- [ ] Tab order is logical

### Testing

- [ ] >80% code coverage on models
- [ ] Golden file tests for main views
- [ ] Integration tests for core workflows

## Implementation Details

### Responsive Layout System

```go
type Layout struct {
    width  int
    height int
    mode   LayoutMode
}

type LayoutMode int

const (
    LayoutCompact  LayoutMode = iota  // 80-99 cols
    LayoutStandard                    // 100-139 cols
    LayoutWide                        // 140+ cols
)

func (l *Layout) Update(width, height int) {
    l.width = width
    l.height = height
    switch {
    case width < 100:
        l.mode = LayoutCompact
    case width < 140:
        l.mode = LayoutStandard
    default:
        l.mode = LayoutWide
    }
}

func (l *Layout) ListPaneWidth() int {
    switch l.mode {
    case LayoutCompact:
        return l.width - 2  // Full width, no detail
    case LayoutStandard:
        return int(float64(l.width) * 0.6)
    case LayoutWide:
        return int(float64(l.width) * 0.5)
    }
    return l.width / 2
}

func (l *Layout) ShowDetailPane() bool {
    return l.mode != LayoutCompact
}
```

### Theme System

```go
type Theme struct {
    Name string

    // Colors
    Primary   lipgloss.Color
    Secondary lipgloss.Color
    Success   lipgloss.Color
    Warning   lipgloss.Color
    Error     lipgloss.Color
    Muted     lipgloss.Color

    // Derived styles (computed from colors)
    Title      lipgloss.Style
    Subtitle   lipgloss.Style
    Selected   lipgloss.Style
    Unselected lipgloss.Style
    Border     lipgloss.Style
    // ... etc
}

var DarkTheme = Theme{
    Name:      "dark",
    Primary:   lipgloss.Color("86"),   // Cyan
    Secondary: lipgloss.Color("99"),   // Magenta
    Success:   lipgloss.Color("78"),   // Green
    Warning:   lipgloss.Color("178"),  // Yellow
    Error:     lipgloss.Color("196"),  // Red
    Muted:     lipgloss.Color("245"),  // Gray
}

var LightTheme = Theme{
    Name:      "light",
    Primary:   lipgloss.Color("27"),   // Blue
    Secondary: lipgloss.Color("127"),  // Purple
    // ... adjusted for light backgrounds
}

var MonochromeTheme = Theme{
    Name:    "mono",
    // All colors same (default terminal)
    // Rely on Bold, Underline, Reverse
}

func (t *Theme) BuildStyles() Styles {
    return Styles{
        Title:    lipgloss.NewStyle().Foreground(t.Primary).Bold(true),
        Selected: lipgloss.NewStyle().Background(t.Primary).Foreground(lipgloss.Color("0")),
        Error:    lipgloss.NewStyle().Foreground(t.Error),
        // ... etc
    }
}
```

### Golden File Testing

```go
func TestDrinksListView_Render(t *testing.T) {
    vm := drinks.NewListViewModel(mockApp())
    vm.SetDrinks([]domain.Drink{
        {Name: "Margarita", Category: "cocktail"},
        {Name: "Mojito", Category: "cocktail"},
    })
    vm.SetSize(100, 30)

    output := vm.View()

    golden := filepath.Join("testdata", "drinks_view.golden")
    if *update {
        os.WriteFile(golden, []byte(output), 0644)
        return
    }

    expected, _ := os.ReadFile(golden)
    if output != string(expected) {
        t.Errorf("View output doesn't match golden file.\nGot:\n%s\nExpected:\n%s",
            output, string(expected))
    }
}
```

### Mouse Event Handling

```go
// In drinks/surfaces/tui/list_vm.go
func (vm *ListViewModel) Update(msg tea.Msg) (ViewModel, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.MouseMsg:
        switch msg.Type {
        case tea.MouseLeft:
            // Check if click is in list area
            if vm.isInListBounds(msg.X, msg.Y) {
                idx := vm.yToListIndex(msg.Y)
                vm.list.Select(idx)
            }
        case tea.MouseWheelUp:
            vm.list.CursorUp()
        case tea.MouseWheelDown:
            vm.list.CursorDown()
        }
    }
    // ... rest of update
}
```

## Notes

### Progressive Enhancement

Mouse and themes are enhancements—the TUI must work fully with keyboard-only and default colors. Test without mouse and
with `NO_COLOR` environment variable.

### Testing Strategy

Golden file tests are fragile to style changes but catch regressions. Use sparingly for stable views. Unit tests for
logic, integration tests for workflows.

### Performance Baselines

Establish performance baselines early in this sprint. Document acceptable thresholds (e.g., "list of 1000 items renders
in <100ms").

### Accessibility Audit

Consider running through all views with a screen reader simulation (text-only output) to verify usability.
