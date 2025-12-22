# Bug Fix: Completion Manager Cursor Navigation

## Issue
When launching `--manage`, the cursor didn't appear at the first quest and didn't wrap around when navigating. It took 6 clicks of DOWN to reach the first visible quest under Celeste, suggesting items 0-5 existed in navigation but weren't visible on screen.

## Root Cause
1. Navigation (up/down) didn't wrap from top to bottom or bottom to top
2. Viewport wasn't being synchronized with cursor position on every update
3. Cursor could land on items outside the visible viewport range

## Changes Made

### 1. Added Cursor Wrapping (`internal/ui/completion.go`)

**UP arrow behavior:**
- Before: Stopped at cursor=0
- After: Wraps to last item when pressing UP at the first item

**DOWN arrow behavior:**
- Before: Stopped at last item
- After: Wraps to cursor=0 when pressing DOWN at the last item

**Code changes (lines 98-132):**
```go
case "up", "k":
    totalItems := m.getTotalItems()
    if totalItems == 0 {
        break
    }
    if m.cursor > 0 {
        m.cursor--
    } else {
        // Wrap to bottom
        m.cursor = totalItems - 1
    }
    // Scroll viewport to keep cursor visible
    if m.cursor < m.viewportTop {
        m.viewportTop = m.cursor
    } else if m.cursor >= m.viewportTop+m.viewportHeight {
        m.viewportTop = max(0, m.cursor-m.viewportHeight+1)
    }

case "down", "j":
    totalItems := m.getTotalItems()
    if totalItems == 0 {
        break
    }
    if m.cursor < totalItems-1 {
        m.cursor++
    } else {
        // Wrap to top
        m.cursor = 0
    }
    // Scroll viewport to keep cursor visible
    if m.cursor < m.viewportTop {
        m.viewportTop = m.cursor
    } else if m.cursor >= m.viewportTop+m.viewportHeight {
        m.viewportTop = max(0, m.cursor-m.viewportHeight+1)
    }
```

### 2. Request Window Size on Init

Changed `Init()` to request window size immediately:
```go
func (m CompletionModel) Init() tea.Cmd {
    return tea.WindowSize()
}
```

This ensures the viewport is properly sized before the first render.

### 3. Validate Initial Cursor Position

Added validation in `NewCompletionModel()`:
```go
// Ensure cursor starts at first valid item
totalItems := model.getTotalItems()
if totalItems > 0 && model.cursor >= totalItems {
    model.cursor = 0
}
```

## Testing

Build test: ✅ Compiles successfully

To test manually:
```bash
go build ./cmd/arcitems
./arcitems --manage
```

Expected behavior:
- Cursor (→) should appear at the first quest immediately
- Pressing DOWN at the bottom wraps to the top
- Pressing UP at the top wraps to the bottom
- Viewport scrolls to keep cursor visible

### 4. Added Defensive Cursor Visibility Check

Added `ensureCursorVisible()` method that's called on every Update cycle:
```go
func (m *CompletionModel) ensureCursorVisible() {
    totalItems := m.getTotalItems()
    if totalItems == 0 {
        m.viewportTop = 0
        return
    }

    // Ensure cursor is within bounds
    if m.cursor < 0 {
        m.cursor = 0
    }
    if m.cursor >= totalItems {
        m.cursor = totalItems - 1
    }

    // Adjust viewport to show cursor
    if m.cursor < m.viewportTop {
        // Cursor is above viewport, scroll up
        m.viewportTop = m.cursor
    } else if m.cursor >= m.viewportTop+m.viewportHeight {
        // Cursor is below viewport, scroll down
        m.viewportTop = max(0, m.cursor-m.viewportHeight+1)
    }
}
```

Called at the end of `Update()` to ensure cursor is always visible regardless of how it got there.

## Files Modified
- `internal/ui/completion.go` (lines 51-73, 98-132, 167-168, 486-510)
