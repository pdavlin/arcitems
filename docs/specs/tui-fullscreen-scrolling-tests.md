# Test Plan: Full-Screen TUI with In-Place Scrolling

## Test Environment Setup

```bash
# Build the binary
go build -o arcitems ./cmd/arcitems

# Ensure test data exists
./arcitems --help  # Should show version and help

# Verify management mode launches
./arcitems --manage  # Should enter TUI
```

## Unit Test Cases

### 1. Viewport Calculation Tests

```go
func TestCalculateViewportHeight(t *testing.T) {
    tests := []struct {
        name           string
        terminalHeight int
        expectedHeight int
    }{
        {"Standard terminal", 24, 17},  // 24 - 4 (header) - 3 (footer)
        {"Large terminal", 60, 53},
        {"Small terminal", 10, 5},  // Minimum enforced
        {"Tiny terminal", 5, 5},    // Minimum enforced
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            m := CompletionModel{height: tt.terminalHeight}
            result := m.calculateViewportHeight()
            if result != tt.expectedHeight {
                t.Errorf("expected %d, got %d", tt.expectedHeight, result)
            }
        })
    }
}
```

### 2. Viewport Scrolling Logic Tests

```go
func TestViewportScrollOnCursorMove(t *testing.T) {
    m := CompletionModel{
        cursor:         10,
        viewportTop:    5,
        viewportHeight: 10,
        quests:         make([]*data.Quest, 50),  // 50 total items
    }

    // Move cursor down beyond viewport
    m.cursor = 16  // Beyond viewportTop (5) + viewportHeight (10) = 15
    m.adjustViewportForCursor()

    if m.viewportTop != 7 {  // Should scroll down: 16 - 10 + 1 = 7
        t.Errorf("expected viewportTop 7, got %d", m.viewportTop)
    }

    // Move cursor up above viewport
    m.cursor = 5
    m.adjustViewportForCursor()

    if m.viewportTop != 5 {  // Should scroll up to show cursor
        t.Errorf("expected viewportTop 5, got %d", m.viewportTop)
    }
}
```

### 3. Visible Item Slice Tests

```go
func TestGetVisibleItemSlice(t *testing.T) {
    quests := make([]*data.Quest, 50)
    for i := range quests {
        quests[i] = &data.Quest{ID: fmt.Sprintf("quest_%d", i)}
    }

    m := CompletionModel{
        quests:         quests,
        viewportTop:    10,
        viewportHeight: 5,
    }

    visible := m.getVisibleItemSlice()

    if len(visible) != 5 {
        t.Errorf("expected 5 visible items, got %d", len(visible))
    }

    // Check correct items are visible
    firstID := visible[0].ID
    if firstID != "quest_10" {
        t.Errorf("expected quest_10, got %s", firstID)
    }
}
```

### 4. Window Resize Tests

```go
func TestWindowResize(t *testing.T) {
    m := CompletionModel{
        cursor:         20,
        viewportTop:    15,
        viewportHeight: 10,
        height:         24,
        quests:         make([]*data.Quest, 50),
    }

    // Resize to smaller height
    m.height = 15
    m.handleWindowResize()

    // Viewport should shrink, cursor should remain visible
    if m.cursor < m.viewportTop || m.cursor >= m.viewportTop+m.viewportHeight {
        t.Error("cursor not visible after resize")
    }

    // Resize to larger height
    m.height = 40
    m.handleWindowResize()

    // Viewport should expand, cursor should still be visible
    if m.cursor < m.viewportTop || m.cursor >= m.viewportTop+m.viewportHeight {
        t.Error("cursor not visible after resize")
    }
}
```

## Integration Tests

### 5. End-to-End Navigation Test

```go
func TestE2ENavigation(t *testing.T) {
    // Setup
    state := &state.CompletionState{
        CompletedQuests:   []string{},
        CompletedProjects: []string{},
        HideoutLevels:     map[string]int{},
    }

    quests := loadTestQuests(50)
    projects := loadTestProjects(10)
    hideouts := loadTestHideouts(5)

    model := NewCompletionModel(state, quests, projects, hideouts)
    model.height = 24
    model.width = 80

    // Simulate navigation
    for i := 0; i < 60; i++ {
        model, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})

        // Verify invariants
        if model.cursor < model.viewportTop {
            t.Errorf("iteration %d: cursor (%d) above viewport (%d)",
                i, model.cursor, model.viewportTop)
        }
        if model.cursor >= model.viewportTop+model.viewportHeight {
            t.Errorf("iteration %d: cursor (%d) below viewport (%d + %d)",
                i, model.cursor, model.viewportTop, model.viewportHeight)
        }
    }
}
```

## Manual Test Cases

### Test Case 1: Basic Alt Screen Behavior

**Objective**: Verify alt screen buffer works correctly

**Steps**:
1. Run some commands in terminal (e.g., `ls`, `echo "test"`)
2. Launch `./arcitems --manage`
3. Verify screen clears completely
4. Navigate around the TUI
5. Press `q` to quit

**Expected Result**:
- TUI takes over full screen
- Previous terminal history restored on exit
- No leftover TUI artifacts in terminal

**Actual Result**: [ PASS / FAIL ]

---

### Test Case 2: Scrolling with Long Lists

**Objective**: Verify viewport scrolling works correctly

**Prerequisites**: Test data with at least 50 quests

**Steps**:
1. Launch `./arcitems --manage`
2. Note terminal height (e.g., 24 lines)
3. Press `j` or down arrow repeatedly
4. Observe list scrolling behavior

**Expected Result**:
- Cursor moves down through list
- When cursor reaches bottom of viewport, list scrolls up
- Footer remains pinned to bottom
- Header remains pinned to top
- Cursor always visible

**Actual Result**: [ PASS / FAIL ]

---

### Test Case 3: Scrolling Indicators

**Objective**: Verify "more above/below" indicators

**Steps**:
1. Launch `./arcitems --manage`
2. Scroll down several items
3. Check for "↑ More above" indicator
4. Scroll to bottom
5. Check that "↓ More below" disappears

**Expected Result**:
- "↑" indicator appears when `viewportTop > 0`
- "↓" indicator appears when items exist below viewport
- Indicators disappear at boundaries

**Actual Result**: [ PASS / FAIL ]

---

### Test Case 4: Window Resize Handling

**Objective**: Verify layout adapts to terminal size changes

**Steps**:
1. Launch `./arcitems --manage` in terminal
2. Scroll to middle of list (cursor at item ~25)
3. Make terminal smaller (drag corner or use keyboard shortcut)
4. Verify cursor still visible
5. Make terminal larger
6. Verify layout expands properly

**Expected Result**:
- Viewport recalculates on resize
- Cursor remains visible at all times
- No visual glitches or artifacts
- Footer stays at bottom

**Actual Result**: [ PASS / FAIL ]

---

### Test Case 5: Section Switching with Viewport

**Objective**: Verify section switching resets viewport appropriately

**Steps**:
1. Launch `./arcitems --manage`
2. Scroll down in Quests section
3. Press `Tab` to switch to Projects
4. Observe viewport behavior
5. Press `Tab` again to switch to Hideouts

**Expected Result**:
- Viewport centers on new section's first item
- Cursor moves to appropriate position in new section
- No jarring visual jumps

**Actual Result**: [ PASS / FAIL ]

---

### Test Case 6: Very Small Terminal

**Objective**: Verify minimum viewport size enforcement

**Steps**:
1. Resize terminal to very small size (e.g., 80x10)
2. Launch `./arcitems --manage`
3. Navigate through list

**Expected Result**:
- Minimum viewport height enforced (5 items)
- Navigation still works
- Footer may overlap content at extreme sizes
- Application doesn't crash

**Actual Result**: [ PASS / FAIL ]

---

### Test Case 7: Very Large Terminal

**Objective**: Verify behavior when list fits entirely in viewport

**Steps**:
1. Resize terminal to large size (e.g., 200x60)
2. Launch `./arcitems --manage`
3. Navigate through list

**Expected Result**:
- All items visible at once
- No scroll indicators
- `viewportTop` remains at 0
- Navigation still works

**Actual Result**: [ PASS / FAIL ]

---

### Test Case 8: Toggle and Save with Scrolling

**Objective**: Verify state management during scrolling

**Steps**:
1. Launch `./arcitems --manage`
2. Scroll to item 30
3. Press `Space` to toggle quest completion
4. Scroll to item 40
5. Press `Space` to toggle another quest
6. Press `s` to save and exit
7. Re-launch `./arcitems --manage`

**Expected Result**:
- State changes persist correctly
- Checkboxes reflect saved state
- Scrolling doesn't interfere with state management

**Actual Result**: [ PASS / FAIL ]

---

## Performance Tests

### Test Case 9: Rendering Performance

**Objective**: Measure frame render time

**Steps**:
1. Add timing instrumentation to `View()` method
2. Launch with 100+ items
3. Scroll rapidly through entire list
4. Measure average render time per frame

**Expected Result**:
- Average render time: < 1ms
- Max render time: < 5ms
- No visible lag or stutter

**Measurement**: _________ ms average, _________ ms max

---

### Test Case 10: Viewport vs Full List Rendering

**Objective**: Compare performance of viewport vs full rendering

**Steps**:
1. Benchmark `View()` with full list rendering (before change)
2. Benchmark `View()` with viewport rendering (after change)
3. Compare results

**Expected Result**:
- Viewport rendering 3-5x faster for long lists
- Memory allocation reduced

**Measurement**:
- Full list: _________ ns/op, _________ allocs/op
- Viewport: _________ ns/op, _________ allocs/op

---

## Cross-Platform Tests

### Test Case 11: macOS Terminal Compatibility

**Platforms**: Terminal.app, iTerm2, Alacritty, Kitty

**Steps**:
1. Test on each terminal emulator
2. Verify alt screen works
3. Verify scrolling behavior
4. Verify exit/restore behavior

**Expected Result**: Identical behavior across all terminals

| Terminal | Alt Screen | Scrolling | Exit/Restore |
|----------|-----------|-----------|--------------|
| Terminal.app | [ ] | [ ] | [ ] |
| iTerm2 | [ ] | [ ] | [ ] |
| Alacritty | [ ] | [ ] | [ ] |
| Kitty | [ ] | [ ] | [ ] |

---

### Test Case 12: Linux Terminal Compatibility

**Platforms**: gnome-terminal, xterm, konsole

**Steps**: Same as Test Case 11

| Terminal | Alt Screen | Scrolling | Exit/Restore |
|----------|-----------|-----------|--------------|
| gnome-terminal | [ ] | [ ] | [ ] |
| xterm | [ ] | [ ] | [ ] |
| konsole | [ ] | [ ] | [ ] |

---

## Regression Tests

### Test Case 13: Search Mode Unchanged

**Objective**: Verify search mode not affected by changes

**Steps**:
1. Run `./arcitems --interactive "medkit"`
2. Verify search TUI works as before
3. Check that it doesn't use alt screen (shouldn't for search)

**Expected Result**: No changes to search mode behavior

**Actual Result**: [ PASS / FAIL ]

---

### Test Case 14: Command-Line Mode Unchanged

**Objective**: Verify non-interactive modes work

**Steps**:
1. Run `./arcitems "medkit"`
2. Run `./arcitems --json "medkit"`
3. Verify output identical to pre-change

**Expected Result**: CLI output unchanged

**Actual Result**: [ PASS / FAIL ]

---

## Test Execution Summary

| Test Case | Status | Notes |
|-----------|--------|-------|
| TC1: Alt Screen | [ ] | |
| TC2: Long Lists | [ ] | |
| TC3: Scroll Indicators | [ ] | |
| TC4: Window Resize | [ ] | |
| TC5: Section Switch | [ ] | |
| TC6: Small Terminal | [ ] | |
| TC7: Large Terminal | [ ] | |
| TC8: Toggle/Save | [ ] | |
| TC9: Render Perf | [ ] | |
| TC10: Viewport Perf | [ ] | |
| TC11: macOS Compat | [ ] | |
| TC12: Linux Compat | [ ] | |
| TC13: Search Mode | [ ] | |
| TC14: CLI Mode | [ ] | |

**Overall Status**: [ READY / IN PROGRESS / BLOCKED ]

**Blocker Notes**: _______________________________

**Test Completed By**: _______________________

**Date**: _______________________
