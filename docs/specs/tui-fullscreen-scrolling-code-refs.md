# Code Reference: Full-Screen TUI Implementation

This document shows exact before/after code changes for implementing full-screen TUI with in-place scrolling.

## Change 1: Add Viewport Fields to Model

**File**: `internal/ui/completion.go:14-24`

### Before
```go
// CompletionModel represents the completion manager UI
type CompletionModel struct {
    completionState *state.CompletionState
    quests          []*data.Quest
    projects        []*data.Project
    hideouts        []*data.Hideout
    cursor          int
    section         string // "quests", "projects", or "hideouts"
    saved           bool
    width           int
    height          int
}
```

### After
```go
// CompletionModel represents the completion manager UI
type CompletionModel struct {
    completionState *state.CompletionState
    quests          []*data.Quest
    projects        []*data.Project
    hideouts        []*data.Hideout
    cursor          int
    section         string // "quests", "projects", or "hideouts"
    saved           bool
    width           int
    height          int
    viewportTop     int  // First visible item index in scrolling region
    viewportHeight  int  // Number of items that fit in viewport
}
```

---

## Change 2: Initialize Viewport in Constructor

**File**: `internal/ui/completion.go:26-47`

### Before
```go
func NewCompletionModel(
    completionState *state.CompletionState,
    quests []*data.Quest,
    projects []*data.Project,
    hideouts []*data.Hideout,
) CompletionModel {
    // Sort quests by ID for consistent display
    sort.Slice(quests, func(i, j int) bool {
        return quests[i].ID < quests[j].ID
    })

    return CompletionModel{
        completionState: completionState,
        quests:          quests,
        projects:        projects,
        hideouts:        hideouts,
        cursor:          0,
        section:         "quests",
        saved:           false,
    }
}
```

### After
```go
func NewCompletionModel(
    completionState *state.CompletionState,
    quests []*data.Quest,
    projects []*data.Project,
    hideouts []*data.Hideout,
) CompletionModel {
    // Sort quests by ID for consistent display
    sort.Slice(quests, func(i, j int) bool {
        return quests[i].ID < quests[j].ID
    })

    return CompletionModel{
        completionState: completionState,
        quests:          quests,
        projects:        projects,
        hideouts:        hideouts,
        cursor:          0,
        section:         "quests",
        saved:           false,
        viewportTop:     0,
        viewportHeight:  15, // Default, will be recalculated on first WindowSizeMsg
    }
}
```

---

## Change 3: Update Navigation to Scroll Viewport

**File**: `internal/ui/completion.go:83-92`

### Before
```go
case "up", "k":
    if m.cursor > 0 {
        m.cursor--
    }

case "down", "j":
    totalItems := m.getTotalItems()
    if m.cursor < totalItems-1 {
        m.cursor++
    }
```

### After
```go
case "up", "k":
    if m.cursor > 0 {
        m.cursor--
        // Scroll viewport up if cursor moves above visible region
        if m.cursor < m.viewportTop {
            m.viewportTop = m.cursor
        }
    }

case "down", "j":
    totalItems := m.getTotalItems()
    if m.cursor < totalItems-1 {
        m.cursor++
        // Scroll viewport down if cursor moves below visible region
        if m.cursor >= m.viewportTop+m.viewportHeight {
            m.viewportTop = m.cursor - m.viewportHeight + 1
        }
    }
```

---

## Change 4: Handle Window Resize Events

**File**: `internal/ui/completion.go:99-102`

### Before
```go
case tea.WindowSizeMsg:
    m.width = msg.Width
    m.height = msg.Height
```

### After
```go
case tea.WindowSizeMsg:
    m.width = msg.Width
    m.height = msg.Height

    // Recalculate viewport height based on new terminal size
    m.viewportHeight = m.calculateViewportHeight()

    // Ensure cursor remains visible after resize
    totalItems := m.getTotalItems()
    if m.cursor < m.viewportTop {
        m.viewportTop = m.cursor
    } else if m.cursor >= m.viewportTop+m.viewportHeight {
        m.viewportTop = max(0, min(m.cursor-m.viewportHeight+1, totalItems-m.viewportHeight))
    }
```

---

## Change 5: Add Helper Functions

**File**: `internal/ui/completion.go` (add after `getTotalItems()`)

### New Functions

```go
// calculateViewportHeight determines how many items can fit in the viewport
// based on terminal height, accounting for header and footer space
func (m CompletionModel) calculateViewportHeight() int {
    const (
        headerLines = 4  // Title line + blank + section header + blank
        footerLines = 3  // Blank + help line 1 + help line 2
        minHeight   = 5  // Minimum viewport height for usability
    )

    availableHeight := m.height - headerLines - footerLines
    if availableHeight < minHeight {
        return minHeight
    }
    return availableHeight
}

// getVisibleItemIndices returns the start and end indices for visible items
func (m CompletionModel) getVisibleItemIndices() (start, end int) {
    totalItems := m.getTotalItems()
    start = m.viewportTop
    end = min(m.viewportTop+m.viewportHeight, totalItems)
    return start, end
}

// Helper: min function
func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}

// Helper: max function
func max(a, b int) int {
    if a > b {
        return a
    }
    return b
}
```

---

## Change 6: Update View() to Render Only Visible Items

**File**: `internal/ui/completion.go:179-304`

This is a larger change. The key modifications are:

### Before (excerpt showing quest rendering)
```go
// Quests section
b.WriteString(titleStyle.Render("Quests"))
b.WriteString("\n")
for i, quest := range m.quests {
    cursor := "  "
    if i == m.cursor {
        cursor = cursorStyle.Render("→ ")
    }

    checkbox := "[ ]"
    if m.completionState.IsQuestCompleted(quest.ID) {
        checkbox = safeStyle.Render("[✓]")
    }

    questName := quest.Name["en"]
    if len(questName) > 50 {
        questName = questName[:47] + "..."
    }

    b.WriteString(fmt.Sprintf("%s%s %s (%s)\n", cursor, checkbox, questName, headerStyle.Render(quest.ID)))
}
```

### After (new viewport-aware rendering)
```go
// Calculate visible range
visibleStart, visibleEnd := m.getVisibleItemIndices()
totalItems := m.getTotalItems()
questCount := len(m.quests)
projectCount := len(m.projects)

// Scroll indicators
if m.viewportTop > 0 {
    b.WriteString(headerStyle.Render("  ↑ More above\n"))
}

// Quests section
currentIdx := 0

b.WriteString(titleStyle.Render("Quests"))
b.WriteString("\n")

for i, quest := range m.quests {
    if currentIdx >= visibleEnd {
        break
    }
    if currentIdx >= visibleStart && currentIdx < visibleEnd {
        cursor := "  "
        if currentIdx == m.cursor {
            cursor = cursorStyle.Render("→ ")
        }

        checkbox := "[ ]"
        if m.completionState.IsQuestCompleted(quest.ID) {
            checkbox = safeStyle.Render("[✓]")
        }

        questName := quest.Name["en"]
        if len(questName) > 50 {
            questName = questName[:47] + "..."
        }

        b.WriteString(fmt.Sprintf("%s%s %s (%s)\n", cursor, checkbox, questName, headerStyle.Render(quest.ID)))
    }
    currentIdx++
}

// Similar changes for Projects and Hideouts sections...

// Scroll indicator at bottom
if m.viewportTop+m.viewportHeight < totalItems {
    b.WriteString(headerStyle.Render("  ↓ More below\n"))
}
```

**Note**: The full implementation would apply this pattern to all three sections (Quests, Projects, Hideouts). For brevity, only the Quests section is shown.

---

## Change 7: Enable Alt Screen in RunCompletion

**File**: `internal/ui/completion.go:318-327`

### Before
```go
// RunCompletion runs the completion manager UI
func RunCompletion(
    completionState *state.CompletionState,
    quests []*data.Quest,
    projects []*data.Project,
    hideouts []*data.Hideout,
) error {
    p := tea.NewProgram(NewCompletionModel(completionState, quests, projects, hideouts))
    _, err := p.Run()
    return err
}
```

### After
```go
// RunCompletion runs the completion manager UI
func RunCompletion(
    completionState *state.CompletionState,
    quests []*data.Quest,
    projects []*data.Project,
    hideouts []*data.Hideout,
) error {
    p := tea.NewProgram(
        NewCompletionModel(completionState, quests, projects, hideouts),
        tea.WithAltScreen(),       // Enable alternate screen buffer
    )
    _, err := p.Run()
    return err
}
```

---

## Complete View() Implementation Example

Here's a more complete example of the viewport-aware `View()` method:

```go
func (m CompletionModel) View() string {
    var b strings.Builder

    // Header (always visible)
    completedQuests := len(m.completionState.CompletedQuests)
    totalQuests := len(m.quests)
    maxedHideouts := 0
    for _, hideout := range m.hideouts {
        if m.completionState.GetHideoutLevel(hideout.ID) >= hideout.MaxLevel {
            maxedHideouts++
        }
    }

    b.WriteString(titleStyle.Render(
        fmt.Sprintf("Completion Manager (%d of %d quests, %d/%d hideouts maxed)",
            completedQuests, totalQuests, maxedHideouts, len(m.hideouts))))
    b.WriteString("\n\n")

    // Calculate visible range
    visibleStart, visibleEnd := m.getVisibleItemIndices()
    totalItems := m.getTotalItems()

    // Top scroll indicator
    if m.viewportTop > 0 {
        b.WriteString(headerStyle.Render("  ↑ More above"))
        b.WriteString("\n\n")
    }

    // Render visible items
    m.renderVisibleItems(&b, visibleStart, visibleEnd)

    // Bottom scroll indicator
    if m.viewportTop+m.viewportHeight < totalItems {
        b.WriteString("\n")
        b.WriteString(headerStyle.Render("  ↓ More below"))
    }

    // Footer (always visible)
    b.WriteString("\n\n")
    b.WriteString(headerStyle.Render("Space: toggle | +/-: adjust level | ↑/↓: navigate | Tab: next section\n"))
    b.WriteString(headerStyle.Render("s: save & exit | q: quit"))

    return b.String()
}

// renderVisibleItems handles the actual rendering of items within the visible range
func (m CompletionModel) renderVisibleItems(b *strings.Builder, visibleStart, visibleEnd int) {
    currentIdx := 0
    questCount := len(m.quests)
    projectCount := len(m.projects)

    // Render Quests
    if currentIdx < visibleEnd && visibleStart < questCount {
        b.WriteString(titleStyle.Render("Quests"))
        b.WriteString("\n")

        for i, quest := range m.quests {
            if currentIdx >= visibleEnd {
                break
            }
            if currentIdx >= visibleStart {
                m.renderQuestItem(b, quest, currentIdx)
            }
            currentIdx++
        }
    } else {
        currentIdx += questCount
    }

    // Render Projects
    if len(m.projects) > 0 && currentIdx < visibleEnd && visibleStart < questCount+projectCount {
        b.WriteString("\n")
        b.WriteString(titleStyle.Render("Projects"))
        b.WriteString("\n")

        for i, project := range m.projects {
            if currentIdx >= visibleEnd {
                break
            }
            if currentIdx >= visibleStart {
                m.renderProjectItem(b, project, currentIdx)
            }
            currentIdx++
        }
    } else {
        currentIdx += projectCount
    }

    // Render Hideouts
    if len(m.hideouts) > 0 && currentIdx < visibleEnd {
        b.WriteString("\n")
        b.WriteString(titleStyle.Render("Hideout Stations"))
        b.WriteString("\n")

        for _, hideout := range m.hideouts {
            if hideout.MaxLevel == 0 {
                continue
            }
            if currentIdx >= visibleEnd {
                break
            }
            if currentIdx >= visibleStart {
                m.renderHideoutItem(b, hideout, currentIdx)
            }
            currentIdx++
        }
    }
}

// renderQuestItem renders a single quest item
func (m CompletionModel) renderQuestItem(b *strings.Builder, quest *data.Quest, idx int) {
    cursor := "  "
    if idx == m.cursor {
        cursor = cursorStyle.Render("→ ")
    }

    checkbox := "[ ]"
    if m.completionState.IsQuestCompleted(quest.ID) {
        checkbox = safeStyle.Render("[✓]")
    }

    questName := quest.Name["en"]
    if len(questName) > 50 {
        questName = questName[:47] + "..."
    }

    b.WriteString(fmt.Sprintf("%s%s %s (%s)\n", cursor, checkbox, questName, headerStyle.Render(quest.ID)))
}

// renderProjectItem renders a single project item
func (m CompletionModel) renderProjectItem(b *strings.Builder, project *data.Project, idx int) {
    cursor := "  "
    if idx == m.cursor {
        cursor = cursorStyle.Render("→ ")
    }

    checkbox := "[ ]"
    if m.completionState.IsProjectCompleted(project.ID) {
        checkbox = safeStyle.Render("[✓]")
    }

    projectName := project.Name["en"]
    if len(projectName) > 50 {
        projectName = projectName[:47] + "..."
    }

    b.WriteString(fmt.Sprintf("%s%s %s\n", cursor, checkbox, projectName))
}

// renderHideoutItem renders a single hideout item
func (m CompletionModel) renderHideoutItem(b *strings.Builder, hideout *data.Hideout, idx int) {
    cursor := "  "
    if idx == m.cursor {
        cursor = cursorStyle.Render("→ ")
    }

    currentLevel := m.completionState.GetHideoutLevel(hideout.ID)
    maxLevel := hideout.MaxLevel

    // Create progress bar
    progressBar := ""
    for j := 1; j <= maxLevel; j++ {
        if j <= currentLevel {
            progressBar += "="
        } else if j == currentLevel+1 {
            progressBar += "●"
        } else {
            progressBar += "-"
        }
    }
    if currentLevel >= maxLevel {
        progressBar = strings.Repeat("=", maxLevel) + "●"
    }

    statusText := fmt.Sprintf("Level %d of %d", currentLevel, maxLevel)
    if currentLevel >= maxLevel {
        statusText += " " + safeStyle.Render("(MAX)")
    }

    stationName := hideout.Name["en"]
    if len(stationName) > 20 {
        stationName = stationName[:17] + "..."
    }
    stationName = fmt.Sprintf("%-20s", stationName)

    b.WriteString(fmt.Sprintf("%s%s [%s] %s\n", cursor, stationName, progressBar, statusText))
}
```

---

## Summary of Changes

| File | Lines Changed | Additions | Deletions | Complexity |
|------|--------------|-----------|-----------|------------|
| `internal/ui/completion.go` | ~150 | ~100 | ~50 | Medium |

### Total Impact
- **Files Modified**: 1
- **New Functions**: 7 helper functions
- **Modified Functions**: 4 existing functions
- **Lines of Code**: ~100 net additions

### Testing Hooks

Add these debug helpers during development:

```go
// Add to CompletionModel for debugging
func (m CompletionModel) DebugViewport() string {
    return fmt.Sprintf("cursor=%d viewport=[%d:%d] height=%d total=%d",
        m.cursor, m.viewportTop, m.viewportTop+m.viewportHeight,
        m.viewportHeight, m.getTotalItems())
}
```

Use in `View()` during development:
```go
b.WriteString(headerStyle.Render(m.DebugViewport()))
b.WriteString("\n")
```

Remove before final commit.

---

## Migration Path

1. **Phase 1**: Add viewport fields and helpers (non-breaking)
2. **Phase 2**: Update navigation logic (functional change)
3. **Phase 3**: Refactor View() to use viewport (visual change)
4. **Phase 4**: Enable alt screen (UX change)

Each phase can be tested independently before proceeding to the next.
