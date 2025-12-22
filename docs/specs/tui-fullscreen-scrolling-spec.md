# Feature Specification: Full-Screen TUI with In-Place Scrolling for Management Mode

## Executive Summary

- **Feature Name**: Full-Screen TUI with In-Place Scrolling
- **Business Value**: Improved UX for managing completion state - handles long lists gracefully, prevents visual jumping
- **Complexity Score**: 3 (Simple)
- **Estimated Effort**: 2-3 hours

### Current Behavior

The management mode (`arcitems --manage`) currently displays all items at once without:
- Taking over the full terminal screen (alt screen buffer)
- Implementing viewport-based scrolling (list scrolls off screen)
- Properly centering the cursor position within the visible viewport

### Target Behavior

- TUI takes over entire terminal screen using alt screen buffer
- List scrolls in place with cursor remaining visible
- Footer stays pinned to bottom of screen
- Smooth navigation without content scrolling past viewport

## Technical Architecture

### Affected Files

- `internal/ui/completion.go:324` - Modify `RunCompletion()` to use alt screen
- `internal/ui/completion.go:179-304` - Update `View()` for viewport-aware rendering
- `internal/ui/completion.go:38-45` - Add viewport tracking to `CompletionModel`

### Dependencies

**Existing (No Changes Required)**:
- `github.com/charmbracelet/bubbletea v1.3.10` - Already supports alt screen via options
- `github.com/charmbracelet/lipgloss v1.1.0` - Styling remains the same

**No New Dependencies Required**

### Implementation Strategy

#### 1. Enable Alt Screen Buffer

The alt screen buffer is a separate terminal screen that allows full takeover without disturbing the user's terminal history.

**Change Location**: `internal/ui/completion.go:324`

```go
// Current
func RunCompletion(...) error {
    p := tea.NewProgram(NewCompletionModel(...))
    _, err := p.Run()
    return err
}

// New
func RunCompletion(...) error {
    p := tea.NewProgram(
        NewCompletionModel(...),
        tea.WithAltScreen(),
    )
    _, err := p.Run()
    return err
}
```

**Rationale**: `tea.WithAltScreen()` enables the alternate screen buffer, which:
- Clears the screen on entry
- Restores previous terminal state on exit
- Provides clean full-screen experience

#### 2. Add Viewport Tracking to Model

**Change Location**: `internal/ui/completion.go:14-24`

```go
type CompletionModel struct {
    completionState *state.CompletionState
    quests          []*data.Quest
    projects        []*data.Project
    hideouts        []*data.Hideout
    cursor          int
    section         string
    saved           bool
    width           int
    height          int
    // ADD THESE:
    viewportTop     int  // First visible item index
    viewportHeight  int  // Number of visible items (calculated from height)
}
```

**Viewport Logic**:
- `viewportTop`: Index of first item visible in scrolling region
- `viewportHeight`: Calculated as `height - headerLines - footerLines - padding`
- Cursor must stay within `[viewportTop, viewportTop + viewportHeight)`

#### 3. Implement Viewport Scrolling in Update()

**Change Location**: `internal/ui/completion.go:83-92`

```go
case "up", "k":
    if m.cursor > 0 {
        m.cursor--
        // NEW: Scroll viewport if cursor moves above visible region
        if m.cursor < m.viewportTop {
            m.viewportTop = m.cursor
        }
    }

case "down", "j":
    totalItems := m.getTotalItems()
    if m.cursor < totalItems-1 {
        m.cursor++
        // NEW: Scroll viewport if cursor moves below visible region
        if m.cursor >= m.viewportTop + m.viewportHeight {
            m.viewportTop = m.cursor - m.viewportHeight + 1
        }
    }
```

**Edge Cases**:
- List shorter than viewport: `viewportTop = 0`, show all items
- Cursor at top: Cannot scroll up further
- Cursor at bottom: Cannot scroll down further
- Window resize: Recalculate `viewportHeight`, adjust `viewportTop` if needed

#### 4. Update View() for Viewport-Aware Rendering

**Change Location**: `internal/ui/completion.go:179-304`

**Header** (always visible):
```
Completion Manager (5 of 10 quests, 3/5 hideouts maxed)

Quests
```

**Scrolling Content Region** (only render visible slice):
```go
// Calculate visible slice
visibleItems := m.getVisibleItemSlice()

// Render only visible items
for i, item := range visibleItems {
    actualIndex := m.viewportTop + i
    cursor := "  "
    if actualIndex == m.cursor {
        cursor = "→ "
    }
    // ... rest of rendering
}
```

**Footer** (always visible, pinned to bottom):
```
Space: toggle | +/-: adjust level | ↑/↓: navigate | Tab: next section
s: save & exit | q: quit
```

**Scroll Indicators**:
- Show "↑ More above" when `viewportTop > 0`
- Show "↓ More below" when `viewportTop + viewportHeight < totalItems`

#### 5. Handle Window Resize

**Change Location**: `internal/ui/completion.go:99-102`

```go
case tea.WindowSizeMsg:
    m.width = msg.Width
    m.height = msg.Height

    // NEW: Recalculate viewport
    m.viewportHeight = m.calculateViewportHeight()

    // NEW: Ensure cursor is still visible
    if m.cursor < m.viewportTop {
        m.viewportTop = m.cursor
    } else if m.cursor >= m.viewportTop + m.viewportHeight {
        m.viewportTop = m.cursor - m.viewportHeight + 1
    }
```

**Helper Function**:
```go
func (m CompletionModel) calculateViewportHeight() int {
    headerLines := 4  // Title + blank + section header + blank
    footerLines := 3  // Blank + help line 1 + help line 2
    sectionHeaderLines := 2  // Per-section headers

    availableHeight := m.height - headerLines - footerLines
    if availableHeight < 5 {
        return 5  // Minimum viewport size
    }
    return availableHeight
}
```

### Visual Layout Calculation

```
┌─────────────────────────────────────┐
│ Header (4 lines)                    │ ← Always visible
├─────────────────────────────────────┤
│                                     │
│ ↑ More above (conditional)          │
│                                     │
│ Scrolling Content (N items)         │ ← Viewport window
│   [viewportTop ... viewportTop+N]   │
│                                     │
│ ↓ More below (conditional)          │
│                                     │
├─────────────────────────────────────┤
│ Footer (3 lines)                    │ ← Always visible
└─────────────────────────────────────┘
```

## Implementation Checklist

### Pre-Implementation
- [x] Review Bubble Tea alt screen documentation
- [x] Analyze existing viewport patterns in similar projects
- [x] Confirm terminal size edge cases (very small terminals)

### Core Implementation (in order)
- [ ] Add viewport fields to `CompletionModel` struct
- [ ] Implement `calculateViewportHeight()` helper
- [ ] Update `NewCompletionModel()` to initialize viewport fields
- [ ] Modify `Update()` to handle viewport scrolling on cursor movement
- [ ] Update `WindowSizeMsg` handler to recalculate viewport
- [ ] Refactor `View()` to render only visible items
- [ ] Add scroll indicators (↑/↓) to View()
- [ ] Update `RunCompletion()` to use `tea.WithAltScreen()`

### Testing
- [ ] Test with list shorter than viewport (no scrolling needed)
- [ ] Test with list longer than viewport (scrolling required)
- [ ] Test window resize while scrolled (cursor remains visible)
- [ ] Test navigation at boundaries (top/bottom)
- [ ] Test section switching maintains viewport position
- [ ] Test on small terminal (80x24)
- [ ] Test on large terminal (200x60)

## Risk Analysis

### Technical Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Off-by-one errors in viewport calculation | Medium | High | Add unit tests for edge cases |
| Cursor jumps on window resize | Low | Medium | Always ensure cursor within viewport after resize |
| Section headers break viewport logic | Low | Low | Count section headers in height calculations |
| Very small terminals (< 10 lines) | Low | Medium | Set minimum viewport height of 5 |

### Breaking Changes

**None** - This is purely a presentation layer enhancement. No API changes, no data format changes.

### Performance Impact

**Minimal** - Viewport reduces rendering from O(N) items to O(viewport_height) items, which is actually a performance improvement for long lists.

## Testing Strategy

### Manual Testing Checklist

1. **Basic Scrolling**
   ```bash
   go build -o arcitems ./cmd/arcitems
   ./arcitems --manage
   # Navigate up/down through entire list
   # Verify cursor stays visible
   # Verify footer stays at bottom
   ```

2. **Window Resize**
   - Start management mode
   - Scroll to middle of list
   - Resize terminal smaller
   - Verify cursor still visible
   - Resize terminal larger
   - Verify layout expands properly

3. **Edge Cases**
   - Navigate to top (first quest)
   - Verify no "↑ More above" indicator
   - Navigate to bottom (last hideout)
   - Verify no "↓ More below" indicator

4. **Section Switching**
   - Press Tab to switch sections
   - Verify viewport resets to show new section's cursor
   - Verify section headers remain visible

### Visual Test Matrix

| Terminal Size | List Length | Expected Behavior |
|--------------|-------------|-------------------|
| 80x24        | 10 items    | No scrolling, all visible |
| 80x24        | 50 items    | Scrolling required, ~15 visible |
| 200x60       | 50 items    | All visible at once |
| 80x10        | 50 items    | Minimum viewport (5), heavy scrolling |

## Success Metrics

### Quantitative
- Viewport rendering performance: < 1ms per frame
- No visual flicker during scrolling
- Cursor position updates: < 16ms (60fps)

### Qualitative
- Management mode "feels" like a proper TUI app
- No terminal history pollution after exit
- Natural navigation experience similar to `less` or `vim`

## Code References

### Similar Implementations

- Bubble Tea examples: [list-fancy](https://github.com/charmbracelet/bubbletea/tree/master/examples/list-fancy)
- Viewport component: Consider using `github.com/charmbracelet/bubbles/viewport` (but not required for this implementation)

### Related Code

- Search mode TUI: `internal/ui/ui.go:180-184` (also needs alt screen in future)
- Model initialization: `internal/ui/completion.go:26-47`
- Cursor navigation: `internal/ui/completion.go:83-97`

## Open Questions

None - specification is ready for implementation.

## Implementation Notes

### Why Not Use `bubbles/viewport`?

The `github.com/charmbracelet/bubbles/viewport` component is designed for continuous text scrolling (like a text editor or log viewer). Our use case is simpler:
- Discrete item list (not continuous text)
- Item-based scrolling (not line-based)
- Custom rendering per item type (quests/projects/hideouts)

Implementing viewport logic directly gives us more control and avoids the overhead of an additional dependency.

### Alternative Screen Buffer Behavior

When `tea.WithAltScreen()` is used:
1. On program start: Terminal switches to alt buffer (screen cleared)
2. During execution: All output goes to alt buffer
3. On program exit: Terminal restores previous buffer (history intact)

This is identical to how `vim`, `less`, and `htop` work.

### Performance Optimization

Current `View()` renders all items (~50-100 lines). With viewport, we render only visible items (~15-20 lines). This is a 3-5x reduction in string building operations per frame.

## Rollout Plan

### Phase 1: Implementation (Day 1)
- Implement all changes in feature branch
- Manual testing on developer machine
- Verify alt screen works correctly

### Phase 2: Testing (Day 1-2)
- Test on multiple terminal emulators (iTerm2, Terminal.app, Alacritty)
- Test on different OS (macOS, Linux)
- Verify no regressions in search mode

### Phase 3: Release (Day 2)
- Merge to main
- Build release binaries
- Update README with new behavior (if applicable)

## Validation Checklist

Before marking complete:
- [ ] No broken file references in spec
- [ ] All code locations verified to exist
- [ ] Implementation order is logical
- [ ] Edge cases documented
- [ ] Success criteria are measurable
- [ ] No TODOs or TBDs in spec

---

**Generated**: 2025-11-24
**Feature Owner**: Management Mode Enhancement
**Target Version**: v0.2.0 (or next minor release)
