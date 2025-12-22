# Viewport Component Refactor

## Summary

Refactored the completion manager to use Bubble Tea's official `viewport` component instead of manually managing scroll position. This fixes the cursor visibility bug and is the proper way to handle scrolling in Bubble Tea apps.

## Changes Made

### 1. Added Dependency
```bash
go get github.com/charmbracelet/bubbles/viewport
```

### 2. Updated CompletionModel Structure
**Before:**
```go
type CompletionModel struct {
    // ...
    viewportTop    int
    viewportHeight int
}
```

**After:**
```go
type CompletionModel struct {
    // ...
    viewport viewport.Model
    ready    bool
}
```

### 3. Simplified Update() Function
- Removed all manual viewport position calculations
- Let the viewport component handle scrolling automatically
- Call `viewport.SetContent()` after every state change
- Let viewport component handle its own Update() messages

### 4. Simplified View() Function
**Before:**
- Manually calculated visible range
- Only rendered items within that range
- Added scroll indicators

**After:**
- Renders header
- Calls `viewport.View()` for scrollable content
- Renders footer
- Viewport handles all scrolling automatically

### 5. New renderContent() Method
Renders ALL content (not just visible items) and returns it as a string. The viewport component handles what's actually displayed.

### 6. Removed Functions
- `calculateViewportHeight()` - viewport handles sizing
- `getVisibleItemIndices()` - viewport handles visibility
- `ensureCursorVisible()` - viewport handles scrolling

## Benefits

1. **Fixes the cursor bug** - Viewport always starts at top, shows all content
2. **Less code** - Removed ~100 lines of manual viewport management
3. **More robust** - Uses tested component instead of custom logic
4. **Proper Bubble Tea pattern** - Standard way to handle scrolling
5. **Better terminal compatibility** - Viewport component handles edge cases

## Testing

```bash
go build ./cmd/arcitems
./arcitems --manage
```

Expected behavior:
- Cursor (→) appears immediately on first Apollo quest
- All quests visible (Apollo, Celeste, Lance, Shani, Tian Wen)
- Smooth scrolling with UP/DOWN
- Wrapping from top to bottom and vice versa
- Works correctly in both WezTerm and Ghostty

## Files Modified
- `internal/ui/completion.go` - Complete refactor to use viewport component
