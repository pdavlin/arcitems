# Completion Tracking Feature - Specification

## Executive Summary

**Feature Name**: Interactive Completion Tracking Mode

**Business Value**: Allows players to mark completed quests, projects, and hideout station upgrades, automatically adjusting item recyclability recommendations based on their actual progression. Eliminates false positives where items are flagged as "needed" but the related quest or upgrade is already complete.

**Complexity Score**: 7/10 (Medium-High)
- Affects 5+ existing files (analyzer, UI, data types, CLI)
- Requires persistent state management (new capability)
- No external database (file-based storage)
- Adds new interactive mode with state mutations
- Backward compatible (existing search mode unchanged)

**Estimated Effort**: 3-4 days
- Day 1: Data layer updates + state persistence design and implementation
- Day 2: Analyzer integration with hideout level filtering
- Day 3: UI with progress bars for hideout levels
- Day 4: CLI integration, testing, documentation

**Data Source Update**: Spec now includes actual hideout station data from RaidTheory/arcraiders-data `hideout/*.json` files. Contains 6 stations with 3 upgrade levels each (weapon_bench, equipment_bench, explosives_bench, med_station, refiner, utility_bench). You'll need to update your data fetching script to bundle these files.

## 1. Problem Analysis

### Current Behavior
The tool flags items as "unsafe to sell" if they're required by ANY quest or project, regardless of completion status. This creates false positives for players who have already completed specific quests.

**Example**:
```bash
$ ./arcitems "first wave tape"
● First Wave Tape ✗
  ⚠ Required by 1 quest(s)
  Safe to sell: NO
```

If the player has already completed the "Broken Monument" quest (ss10h), the tape is actually safe to sell, but the tool doesn't know this.

### Desired Behavior
After marking "Broken Monument" as complete:
```bash
$ ./arcitems "first wave tape"
● First Wave Tape ✓
  ✓ Safe to sell/recycle
  Previously required by: Broken Monument (completed)
```

### Design Constraints
1. **Offline-first**: No cloud sync, state stored locally
2. **Backward compatible**: Existing search mode must work unchanged
3. **Non-intrusive**: Completion tracking is opt-in
4. **Fast**: State checks add < 5ms to search operations
5. **Portable**: State file uses standard format (JSON)

## 2. Technical Architecture

### State Storage Design

**Location**: `~/.arcitems/completion.json`

**Schema**:
```json
{
  "version": 1,
  "dataVersion": "2025.11.21.2052",
  "updatedAt": "2025-11-21T20:52:39Z",
  "completedQuests": ["ss10h", "ss5", "ss7h"],
  "completedProjects": ["expedition_project"],
  "hideoutLevels": {
    "weapon_bench": 2,
    "equipment_bench": 3,
    "explosives_bench": 1,
    "med_station": 0,
    "refiner": 0,
    "utility_bench": 0
  }
}
```

**Rationale**:
- Flat array for quests/projects (simpler, faster lookup)
- `hideoutLevels` stores completed level per station (integer 0-3)
- Level 2 means "levels 1 and 2 complete, level 3 not started"
- Level 0 means "no upgrades completed yet"
- `dataVersion` tracks which game data this state applies to
- `version` allows schema evolution

**Hideout Stations**:
Based on RaidTheory/arcraiders-data `hideout/*.json`:
- weapon_bench (maxLevel: 3)
- equipment_bench (maxLevel: 3)
- explosives_bench (maxLevel: 3)
- med_station (maxLevel: 3)
- refiner (maxLevel: 3)
- utility_bench (maxLevel: 3)
- workbench (maxLevel: 0, no upgrades defined)

### Data Migration Strategy

When game data updates (new quests added), handle gracefully:

```go
// Pseudo-logic
if stateFile.dataVersion != currentDataVersion {
    // Option A: Preserve existing completions (questIDs are stable)
    // Option B: Warn user and validate each quest still exists
    // Recommendation: Option A - questIDs don't change

    stateFile.dataVersion = currentDataVersion
    saveState()
}
```

### Architecture Diagram

```
┌──────────────────────────────────────────────────┐
│              CLI Entry Point                     │
│   ./arcitems "query" [--mode=interactive]       │
└───────────────────┬──────────────────────────────┘
                    │
                    ▼
         ┌──────────────────────┐
         │   Load State         │
         │ (~/.arcitems/*.json) │
         └──────────┬───────────┘
                    │
                    ▼
         ┌──────────────────────┐
         │   Analyzer (Modified)│
         │  + CompletionState   │
         └──────────┬───────────┘
                    │
                    ▼
         ┌──────────────────────┐
         │   Search Results     │
         │  (filtered by state) │
         └──────────┬───────────┘
                    │
        ┌───────────┴──────────┐
        │                      │
        ▼                      ▼
┌───────────────┐    ┌─────────────────────┐
│  Normal Mode  │    │  Interactive Mode   │
│  (read-only)  │    │  (stateful)         │
└───────────────┘    └──────────┬──────────┘
                                 │
                                 ▼
                     ┌───────────────────────┐
                     │   Checkbox UI         │
                     │  - Toggle completions │
                     │  - Save on change     │
                     └───────────────────────┘
```

## 3. Data Model Changes

### Hideout Data Types

First, add hideout types to the existing data package:

```go
// internal/data/types.go (additions)

// Hideout represents a hideout station with upgradeable levels
type Hideout struct {
    ID       string            `json:"id"`
    Name     map[string]string `json:"name"`
    MaxLevel int               `json:"maxLevel"`
    Levels   []HideoutLevel    `json:"levels"`
}

// HideoutLevel represents a single upgrade level for a hideout station
type HideoutLevel struct {
    Level              int               `json:"level"`
    RequirementItemIds []ItemRequirement `json:"requirementItemIds"`
}
```

**Data Source**: `hideout/*.json` from RaidTheory/arcraiders-data

**Example** (weapon_bench.json):
```json
{
  "id": "weapon_bench",
  "name": { "en": "Weapon Bench", ... },
  "maxLevel": 3,
  "levels": [
    {
      "level": 1,
      "requirementItemIds": [
        { "itemId": "metal_parts", "quantity": 20 },
        { "itemId": "rubber_parts", "quantity": 30 }
      ]
    },
    ...
  ]
}
```

### State Management Types

```go
// internal/state/state.go

package state

import (
    "encoding/json"
    "os"
    "path/filepath"
    "time"
)

// CompletionState tracks user progress
type CompletionState struct {
    Version           int            `json:"version"`
    DataVersion       string         `json:"dataVersion"`
    UpdatedAt         time.Time      `json:"updatedAt"`
    CompletedQuests   []string       `json:"completedQuests"`
    CompletedProjects []string       `json:"completedProjects"`
    HideoutLevels     map[string]int `json:"hideoutLevels"` // stationID -> completed level (0-3)
}

// NewCompletionState creates a default empty state
func NewCompletionState(dataVersion string) *CompletionState {
    return &CompletionState{
        Version:           1,
        DataVersion:       dataVersion,
        UpdatedAt:         time.Now(),
        CompletedQuests:   []string{},
        CompletedProjects: []string{},
        HideoutLevels:     make(map[string]int),
    }
}

// LoadState loads completion state from disk
func LoadState(dataVersion string) (*CompletionState, error) {
    path := getStatePath()

    if _, err := os.Stat(path); os.IsNotExist(err) {
        // No state file yet, return empty state
        return NewCompletionState(dataVersion), nil
    }

    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }

    var state CompletionState
    if err := json.Unmarshal(data, &state); err != nil {
        return nil, err
    }

    // Handle data version mismatch
    if state.DataVersion != dataVersion {
        // Preserve existing completions but update version
        state.DataVersion = dataVersion
        state.UpdatedAt = time.Now()
        // Note: questIDs are stable, so old completions remain valid
    }

    return &state, nil
}

// SaveState writes completion state to disk
func (s *CompletionState) SaveState() error {
    path := getStatePath()

    // Ensure directory exists
    dir := filepath.Dir(path)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return err
    }

    s.UpdatedAt = time.Now()

    data, err := json.MarshalIndent(s, "", "  ")
    if err != nil {
        return err
    }

    return os.WriteFile(path, data, 0644)
}

// IsQuestCompleted checks if a quest is marked complete
func (s *CompletionState) IsQuestCompleted(questID string) bool {
    for _, id := range s.CompletedQuests {
        if id == questID {
            return true
        }
    }
    return false
}

// ToggleQuest toggles completion status for a quest
func (s *CompletionState) ToggleQuest(questID string) {
    for i, id := range s.CompletedQuests {
        if id == questID {
            // Remove if already present
            s.CompletedQuests = append(s.CompletedQuests[:i], s.CompletedQuests[i+1:]...)
            return
        }
    }
    // Add if not present
    s.CompletedQuests = append(s.CompletedQuests, questID)
}

// IsProjectCompleted checks if a project is marked complete
func (s *CompletionState) IsProjectCompleted(projectID string) bool {
    for _, id := range s.CompletedProjects {
        if id == projectID {
            return true
        }
    }
    return false
}

// ToggleProject toggles completion status for a project
func (s *CompletionState) ToggleProject(projectID string) {
    for i, id := range s.CompletedProjects {
        if id == projectID {
            s.CompletedProjects = append(s.CompletedProjects[:i], s.CompletedProjects[i+1:]...)
            return
        }
    }
    s.CompletedProjects = append(s.CompletedProjects, projectID)
}

// GetHideoutLevel returns the completed level for a hideout station (0 if not started)
func (s *CompletionState) GetHideoutLevel(stationID string) int {
    return s.HideoutLevels[stationID]
}

// SetHideoutLevel sets the completed level for a hideout station
func (s *CompletionState) SetHideoutLevel(stationID string, level int) {
    if level < 0 {
        level = 0
    }
    s.HideoutLevels[stationID] = level
}

// IncrementHideoutLevel increases the hideout level by 1 (up to maxLevel)
func (s *CompletionState) IncrementHideoutLevel(stationID string, maxLevel int) {
    current := s.HideoutLevels[stationID]
    if current < maxLevel {
        s.HideoutLevels[stationID] = current + 1
    }
}

// DecrementHideoutLevel decreases the hideout level by 1 (down to 0)
func (s *CompletionState) DecrementHideoutLevel(stationID string) {
    current := s.HideoutLevels[stationID]
    if current > 0 {
        s.HideoutLevels[stationID] = current - 1
    }
}

// IsHideoutLevelCompleted checks if a specific hideout level is completed
func (s *CompletionState) IsHideoutLevelCompleted(stationID string, level int) bool {
    return s.HideoutLevels[stationID] >= level
}

func getStatePath() string {
    home, _ := os.UserHomeDir()
    return filepath.Join(home, ".arcitems", "completion.json")
}
```

### Modified Analyzer

First, update the Analyzer struct and ItemUsage to track hideout requirements:

```go
// internal/analyzer/analyzer.go (modifications)

// ItemUsage contains analysis results for an item
type ItemUsage struct {
    Item             *data.Item
    UsedInQuests     []string            // Quest IDs that require this item
    UsedInProjects   []string            // Project IDs that require this item
    UsedInHideouts   map[string][]int    // HideoutID -> []level numbers (e.g., "weapon_bench" -> [1, 2])
    SafeToSell       bool
}

// Analyzer analyzes quest, project, and hideout requirements
type Analyzer struct {
    items    map[string]*data.Item
    quests   []*data.Quest
    projects []*data.Project
    hideouts []*data.Hideout  // NEW

    // Caches
    questUsageMap   map[string][]string              // itemID -> []questID
    projectUsageMap map[string][]string              // itemID -> []projectID
    hideoutUsageMap map[string]map[string][]int      // itemID -> hideoutID -> []levels
}

// NewAnalyzer creates a new analyzer with the given data
func NewAnalyzer(
    items map[string]*data.Item,
    quests []*data.Quest,
    projects []*data.Project,
    hideouts []*data.Hideout,  // NEW
) *Analyzer {
    a := &Analyzer{
        items:           items,
        quests:          quests,
        projects:        projects,
        hideouts:        hideouts,
        questUsageMap:   make(map[string][]string),
        projectUsageMap: make(map[string][]string),
        hideoutUsageMap: make(map[string]map[string][]int),
    }
    a.buildUsageIndex()
    return a
}

// buildUsageIndex pre-computes which items are used in quests, projects, and hideouts
func (a *Analyzer) buildUsageIndex() {
    // Build quest usage map (existing logic)
    for _, quest := range a.quests {
        for _, req := range quest.RequiredItemIds {
            a.questUsageMap[req.ItemID] = append(a.questUsageMap[req.ItemID], quest.ID)
        }
    }

    // Build project usage map (existing logic)
    for _, project := range a.projects {
        for _, phase := range project.Phases {
            for _, req := range phase.RequirementItemIds {
                a.projectUsageMap[req.ItemID] = append(a.projectUsageMap[req.ItemID], project.ID)
            }
            // Category requirements...
        }
    }

    // Build hideout usage map (NEW)
    for _, hideout := range a.hideouts {
        for _, level := range hideout.Levels {
            for _, req := range level.RequirementItemIds {
                if a.hideoutUsageMap[req.ItemID] == nil {
                    a.hideoutUsageMap[req.ItemID] = make(map[string][]int)
                }
                a.hideoutUsageMap[req.ItemID][hideout.ID] = append(
                    a.hideoutUsageMap[req.ItemID][hideout.ID],
                    level.Level,
                )
            }
        }
    }
}

// AnalyzeItemWithState returns usage info considering completion state
func (a *Analyzer) AnalyzeItemWithState(itemID string, state *state.CompletionState) *ItemUsage {
    usage := a.AnalyzeItem(itemID) // Use existing analysis
    if usage == nil {
        return nil
    }

    // Filter out completed quests
    var activeQuests []string
    for _, questID := range usage.UsedInQuests {
        if state == nil || !state.IsQuestCompleted(questID) {
            activeQuests = append(activeQuests, questID)
        }
    }

    // Filter out completed projects
    var activeProjects []string
    for _, projectID := range usage.UsedInProjects {
        if state == nil || !state.IsProjectCompleted(projectID) {
            activeProjects = append(activeProjects, projectID)
        }
    }

    // Filter out completed hideout levels
    activeHideouts := make(map[string][]int)
    for hideoutID, levels := range usage.UsedInHideouts {
        completedLevel := 0
        if state != nil {
            completedLevel = state.GetHideoutLevel(hideoutID)
        }

        // Only include levels higher than completed
        var activeLevels []int
        for _, level := range levels {
            if level > completedLevel {
                activeLevels = append(activeLevels, level)
            }
        }

        if len(activeLevels) > 0 {
            activeHideouts[hideoutID] = activeLevels
        }
    }

    // Recalculate safety
    safeToSell := len(activeQuests) == 0 &&
                  len(activeProjects) == 0 &&
                  len(activeHideouts) == 0

    return &ItemUsage{
        Item:           usage.Item,
        UsedInQuests:   activeQuests,
        UsedInProjects: activeProjects,
        UsedInHideouts: activeHideouts,
        SafeToSell:     safeToSell,
    }
}
```

## 4. Interactive UI Implementation

### UI Mode: Completion Manager

**Activation**: `./arcitems --manage` or `./arcitems -m`

**Layout**:
```
┌────────────────────────────────────────────────────────────┐
│ Completion Manager (22 of 66 quests, 2/6 hideouts maxed)  │
│                                                            │
│ Quests                                                     │
│   [x] A Bad Feeling (ss5)                                  │
│   [ ] Broken Monument (ss10h)                              │
│   [x] Communication Problem (ss7h)                         │
│   ...                                                      │
│                                                            │
│ Projects                                                   │
│   [x] Expedition Project (6 phases)                        │
│                                                            │
│ Hideout Stations                                           │
│   → Weapon Bench           [==●-] Level 2 of 3             │
│     Equipment Bench        [===●] Level 3 of 3 (MAX)       │
│     Explosives Bench       [●---] Level 1 of 3             │
│     Med Station            [----] Level 0 of 3             │
│     Refiner                [----] Level 0 of 3             │
│     Utility Bench          [----] Level 0 of 3             │
│                                                            │
│ Space: toggle | +/-: adjust level | ↑/↓: navigate         │
│ s: save & exit | q: quit                                   │
└────────────────────────────────────────────────────────────┘
```

**Navigation**:
- **Space/Enter**: Toggle quest/project completion
- **+/Right**: Increment hideout level (when on hideout station)
- **-/Left**: Decrement hideout level (when on hideout station)
- **↑/↓ or j/k**: Move cursor
- **Tab**: Cycle between sections
- **s**: Save and exit
- **q/Esc**: Quit (warn if unsaved)

### UI Model (Bubble Tea)

```go
// internal/ui/completion.go

package ui

import (
    tea "github.com/charmbracelet/bubbletea"
    "github.com/pdavlin/arcitems/internal/data"
    "github.com/pdavlin/arcitems/internal/state"
)

type CompletionModel struct {
    state    *state.CompletionState
    quests   []*data.Quest
    projects []*data.Project
    hideouts []*data.Hideout  // NEW
    cursor   int
    mode     string // "quests", "projects", "hideouts"
    saved    bool
}

func NewCompletionModel(
    state *state.CompletionState,
    quests []*data.Quest,
    projects []*data.Project,
    hideouts []*data.Hideout,  // NEW
) CompletionModel {
    return CompletionModel{
        state:    state,
        quests:   quests,
        projects: projects,
        hideouts: hideouts,
        cursor:   0,
        mode:     "quests",
        saved:    false,
    }
}

func (m CompletionModel) Init() tea.Cmd {
    return nil
}

func (m CompletionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "q", "esc":
            if !m.saved {
                // Warn about unsaved changes?
            }
            return m, tea.Quit

        case "s":
            // Save and exit
            if err := m.state.SaveState(); err != nil {
                // Handle error
            }
            m.saved = true
            return m, tea.Quit

        case " ", "enter":
            // Toggle current item (quests/projects only)
            m.toggleCurrent()

        case "+", "right", "=":
            // Increment hideout level
            m.incrementHideoutLevel()

        case "-", "left":
            // Decrement hideout level
            m.decrementHideoutLevel()

        case "up", "k":
            if m.cursor > 0 {
                m.cursor--
            }

        case "down", "j":
            if m.cursor < m.getTotalItems()-1 {
                m.cursor++
            }

        case "tab":
            // Switch between sections
            m.nextMode()
        }
    }
    return m, nil
}

func (m *CompletionModel) toggleCurrent() {
    switch m.mode {
    case "quests":
        if m.cursor < len(m.quests) {
            quest := m.quests[m.cursor]
            m.state.ToggleQuest(quest.ID)
        }
    case "projects":
        idx := m.cursor - len(m.quests)
        if idx >= 0 && idx < len(m.projects) {
            project := m.projects[idx]
            m.state.ToggleProject(project.ID)
        }
    }
}

func (m *CompletionModel) incrementHideoutLevel() {
    if m.mode != "hideouts" {
        return
    }

    idx := m.cursor - len(m.quests) - len(m.projects)
    if idx >= 0 && idx < len(m.hideouts) {
        hideout := m.hideouts[idx]
        m.state.IncrementHideoutLevel(hideout.ID, hideout.MaxLevel)
    }
}

func (m *CompletionModel) decrementHideoutLevel() {
    if m.mode != "hideouts" {
        return
    }

    idx := m.cursor - len(m.quests) - len(m.projects)
    if idx >= 0 && idx < len(m.hideouts) {
        hideout := m.hideouts[idx]
        m.state.DecrementHideoutLevel(hideout.ID)
    }
}

func (m CompletionModel) View() string {
    var b strings.Builder

    // Header
    completedCount := len(m.state.CompletedQuests)
    totalCount := len(m.quests)
    b.WriteString(titleStyle.Render(
        fmt.Sprintf("Completion Manager (%d of %d quests completed)\n\n",
            completedCount, totalCount)))

    // Quests section
    b.WriteString(headerStyle.Render("Quests\n"))
    for i, quest := range m.quests {
        cursor := "  "
        if i == m.cursor {
            cursor = cursorStyle.Render("→ ")
        }

        checkbox := "[ ]"
        if m.state.IsQuestCompleted(quest.ID) {
            checkbox = safeStyle.Render("[✓]")
        }

        questName := quest.Name["en"]
        b.WriteString(fmt.Sprintf("%s%s %s (%s)\n", cursor, checkbox, questName, quest.ID))
    }

    // Projects section
    b.WriteString("\n")
    b.WriteString(headerStyle.Render("Projects\n"))
    for i, project := range m.projects {
        idx := len(m.quests) + i
        cursor := "  "
        if idx == m.cursor {
            cursor = cursorStyle.Render("→ ")
        }

        checkbox := "[ ]"
        if m.state.IsProjectCompleted(project.ID) {
            checkbox = safeStyle.Render("[✓]")
        }

        projectName := project.Name["en"]
        b.WriteString(fmt.Sprintf("%s%s %s\n", cursor, checkbox, projectName))
    }

    // Hideout Stations section
    b.WriteString("\n")
    b.WriteString(headerStyle.Render("Hideout Stations\n"))
    for i, hideout := range m.hideouts {
        idx := len(m.quests) + len(m.projects) + i
        cursor := "  "
        if idx == m.cursor {
            cursor = cursorStyle.Render("→ ")
        } else {
            cursor = "  "
        }

        currentLevel := m.state.GetHideoutLevel(hideout.ID)
        maxLevel := hideout.MaxLevel

        // Create progress bar [==●-] style
        progressBar := ""
        for j := 1; j <= maxLevel; j++ {
            if j <= currentLevel {
                progressBar += "="
            } else if j == currentLevel + 1 {
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

        stationName := fmt.Sprintf("%-20s", hideout.Name["en"])
        b.WriteString(fmt.Sprintf("%s%s [%s] %s\n", cursor, stationName, progressBar, statusText))
    }

    // Footer
    b.WriteString("\n")
    b.WriteString(headerStyle.Render("Space: toggle | +/-: adjust level | ↑/↓: navigate\n"))
    b.WriteString(headerStyle.Render("s: save & exit | q: quit"))

    return b.String()
}

func (m CompletionModel) getTotalItems() int {
    return len(m.quests) + len(m.projects) + len(m.hideouts)
}

func (m *CompletionModel) nextMode() {
    switch m.mode {
    case "quests":
        if len(m.projects) > 0 {
            m.mode = "projects"
            m.cursor = len(m.quests)
        } else if len(m.hideouts) > 0 {
            m.mode = "hideouts"
            m.cursor = len(m.quests)
        }
    case "projects":
        if len(m.hideouts) > 0 {
            m.mode = "hideouts"
            m.cursor = len(m.quests) + len(m.projects)
        } else {
            m.mode = "quests"
            m.cursor = 0
        }
    case "hideouts":
        m.mode = "quests"
        m.cursor = 0
    }
}

// RunCompletion runs the completion manager UI
func RunCompletion(
    state *state.CompletionState,
    quests []*data.Quest,
    projects []*data.Project,
    hideouts []*data.Hideout,  // NEW
) error {
    p := tea.NewProgram(NewCompletionModel(state, quests, projects, hideouts))
    _, err := p.Run()
    return err
}
```

## 5. CLI Integration

### Command Structure

```go
// cmd/arcitems/main.go (modifications)

var (
    jsonOutput     bool
    lang           string
    manageMode     bool  // NEW
    noStateFlag    bool  // NEW: disable state loading
)

func init() {
    rootCmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output in JSON format")
    rootCmd.Flags().StringVarP(&lang, "lang", "l", "en", "Search language")
    rootCmd.Flags().BoolVarP(&manageMode, "manage", "m", false, "Launch completion manager")
    rootCmd.Flags().BoolVar(&noStateFlag, "no-state", false, "Ignore completion state (search all items)")
}

func run(cmd *cobra.Command, args []string) error {
    // Load embedded data
    items, quests, projects, hideouts, metadata := data.LoadEmbedded()  // NEW: load hideouts

    // Load completion state (unless disabled)
    var completionState *state.CompletionState
    if !noStateFlag {
        var err error
        completionState, err = state.LoadState(metadata.Version)
        if err != nil {
            // Non-fatal: warn and continue without state
            fmt.Fprintf(os.Stderr, "Warning: could not load state: %v\n", err)
        }
    }

    // If manage mode, launch completion UI
    if manageMode {
        return ui.RunCompletion(completionState, quests, projects, hideouts)  // NEW: pass hideouts
    }

    // Normal search mode with state filtering
    query := strings.Join(args, " ")
    analyzer := analyzer.NewAnalyzer(items, quests, projects, hideouts)  // NEW: pass hideouts
    searcher := search.NewSearcher(items, analyzer, lang)
    results := searcher.Search(query)

    // Apply state filtering to results
    if completionState != nil {
        results = filterResultsByState(results, analyzer, completionState)
    }

    // Display results
    if jsonOutput {
        return outputJSON(results)
    }
    return ui.Run(query, results)
}

func filterResultsByState(
    results []*search.SearchResult,
    analyzer *analyzer.Analyzer,
    state *state.CompletionState,
) []*search.SearchResult {
    filtered := make([]*search.SearchResult, len(results))
    for i, result := range results {
        // Re-analyze with state
        usage := analyzer.AnalyzeItemWithState(result.Usage.Item.ID, state)
        filtered[i] = &search.SearchResult{
            Usage:    usage,
            Score:    result.Score,
            MatchStr: result.MatchStr,
        }
    }
    return filtered
}
```

### Usage Examples

```bash
# Normal search (considers completion state if present)
./arcitems "first wave tape"

# Search ignoring completion state (show ALL quest items)
./arcitems "first wave tape" --no-state

# Launch completion manager
./arcitems --manage

# Launch completion manager (short flag)
./arcitems -m
```

## 6. Enhanced Search Display

### Modified Result Display

When an item was previously required but the quest is complete:

```
$ ./arcitems "first wave tape"

Search: first wave tape

Found 1 match(es)

● First Wave Tape ✓
  Rare | Quest Item | 500 coins
  ✓ Safe to sell/recycle
  Previously required by: Broken Monument (completed)
  Recycles into: 3x paper, 1x plastic

↑/k: up | ↓/j: down | m: manage completions | q: quit
```

**Key changes**:
- Shows "Previously required by" instead of "Required by"
- Indicates "(completed)" status
- Item now shows ✓ instead of ✗

### UI Code Changes

```go
// internal/ui/ui.go (View method modifications)

// In the View() method, after displaying quest usage:
if len(result.Usage.UsedInQuests) > 0 {
    b.WriteString(unsafeStyle.Render(
        fmt.Sprintf("  ⚠ Required by %d quest(s)\n", len(result.Usage.UsedInQuests))))
} else if len(result.Usage.UsedInProjects) > 0 {
    b.WriteString(unsafeStyle.Render(
        fmt.Sprintf("  ⚠ Required by %d project(s)\n", len(result.Usage.UsedInProjects))))
} else {
    // NEW: Check if item was previously required
    allQuests := m.analyzer.GetQuestsForItem(item.ID) // Need to add this method
    if len(allQuests) > 0 {
        // Item was required, but all quests are complete
        b.WriteString(safeStyle.Render("  ✓ Safe to sell/recycle\n"))
        b.WriteString(headerStyle.Render(
            fmt.Sprintf("  Previously required by: %s (completed)\n",
                formatQuestNames(allQuests))))
    } else {
        b.WriteString(safeStyle.Render("  ✓ Safe to sell/recycle\n"))
    }
}
```

## 7. Testing Strategy

### Unit Tests

```go
// internal/state/state_test.go

func TestLoadState_NewFile(t *testing.T) {
    // Should create empty state when no file exists
    state, err := LoadState("2025.11.21.2052")
    if err != nil {
        t.Fatal(err)
    }
    if len(state.CompletedQuests) != 0 {
        t.Error("Expected empty quest list")
    }
}

func TestToggleQuest(t *testing.T) {
    state := NewCompletionState("2025.11.21.2052")

    // Add quest
    state.ToggleQuest("ss10h")
    if !state.IsQuestCompleted("ss10h") {
        t.Error("Quest should be marked complete")
    }

    // Remove quest
    state.ToggleQuest("ss10h")
    if state.IsQuestCompleted("ss10h") {
        t.Error("Quest should not be complete")
    }
}

func TestAnalyzeWithState(t *testing.T) {
    items := map[string]*data.Item{
        "first_wave_tape": {
            ID:   "first_wave_tape",
            Name: map[string]string{"en": "First Wave Tape"},
        },
    }

    quests := []*data.Quest{
        {
            ID:   "ss10h",
            Name: map[string]string{"en": "Broken Monument"},
            RequiredItemIds: []data.ItemRequirement{
                {ItemID: "first_wave_tape", Quantity: 1},
            },
        },
    }

    analyzer := analyzer.NewAnalyzer(items, quests, nil)

    // Without state: item is unsafe
    usage := analyzer.AnalyzeItem("first_wave_tape")
    if usage.SafeToSell {
        t.Error("Item should be unsafe without state")
    }

    // With completed quest: item is safe
    state := NewCompletionState("2025.11.21.2052")
    state.ToggleQuest("ss10h")
    usage = analyzer.AnalyzeItemWithState("first_wave_tape", state)
    if !usage.SafeToSell {
        t.Error("Item should be safe with completed quest")
    }
}
```

### Integration Tests

```bash
# Test 1: Fresh install (no state file)
./arcitems "first wave tape"
# Expected: Shows as unsafe

# Test 2: Mark quest complete
./arcitems --manage
# (Toggle "Broken Monument" quest, press 's')

# Test 3: Search again
./arcitems "first wave tape"
# Expected: Shows as safe, with "Previously required by" message

# Test 4: Search with --no-state
./arcitems "first wave tape" --no-state
# Expected: Shows as unsafe (ignores completion state)

# Test 5: Data version mismatch
# (Manually edit ~/.arcitems/completion.json to have old dataVersion)
./arcitems "battery"
# Expected: Works normally, updates dataVersion silently
```

### Edge Cases

1. **State file corrupted**
   - Behavior: Warn user, fall back to empty state
   - Test: Corrupt JSON in completion.json

2. **Quest removed from game data**
   - Behavior: Silently ignore non-existent questID in state
   - Test: Manually add fake questID to completion.json

3. **Disk full during save**
   - Behavior: Show error, don't corrupt existing file
   - Test: Mock os.WriteFile to return ENOSPC

4. **Concurrent state modifications**
   - Risk: Low (single-user CLI tool)
   - Mitigation: File locking not needed for MVP

## 8. Implementation Checklist

### Phase 0: Data Layer Updates (Day 1)
- [ ] Update `scripts/fetch_data.py` to fetch `hideout/*.json` files
- [ ] Bundle hideout data into `internal/data/bundled/hideouts.json`
- [ ] Add Hideout and HideoutLevel types to `internal/data/types.go`
- [ ] Update `internal/data/embedded.go` to embed hideouts.json
- [ ] Update LoadEmbedded() to return hideouts slice
- [ ] Verify all 7 hideout stations load correctly (6 with levels, workbench with maxLevel: 0)
- [ ] Write unit tests for hideout data loading

### Phase 1: State Management (Day 1)
- [ ] Create `internal/state/state.go` package
- [ ] Implement CompletionState struct and methods
- [ ] Implement LoadState / SaveState with error handling
- [ ] Add data version migration logic
- [ ] Write unit tests for state operations
- [ ] Test state file creation in ~/.arcitems/

### Phase 2: Analyzer Integration (Day 2)
- [ ] Update Analyzer struct to include hideouts parameter
- [ ] Add hideoutUsageMap to buildUsageIndex()
- [ ] Update ItemUsage struct to include UsedInHideouts field
- [ ] Add AnalyzeItemWithState method with hideout level filtering
- [ ] Implement quest/project/hideout filtering logic
- [ ] Add helper methods (GetQuestsForItem, GetHideoutByID, etc.)
- [ ] Write unit tests for hideout level filtering
- [ ] Test performance impact (target < 5ms)

### Phase 3: Interactive UI (Day 2-3)
- [ ] Create `internal/ui/completion.go`
- [ ] Implement CompletionModel with Bubble Tea
- [ ] Add checkbox rendering for quests/projects
- [ ] Add progress bar rendering for hideout levels
- [ ] Implement toggle logic for quests/projects
- [ ] Implement increment/decrement for hideout levels
- [ ] Add +/- and left/right key handlers
- [ ] Implement save functionality
- [ ] Handle navigation (up/down, space, tab)
- [ ] Test keyboard interactions manually

### Phase 4: CLI Integration (Day 3)
- [ ] Add --manage flag to main.go
- [ ] Add --no-state flag
- [ ] Integrate state loading in search flow
- [ ] Update UI to show "Previously required by" messages
- [ ] Add completion manager to footer hint
- [ ] Test all CLI flag combinations

### Phase 5: Documentation & Polish (Day 4)
- [ ] Update README with new features
- [ ] Add examples for completion tracking
- [ ] Document state file format
- [ ] Test data version migration
- [ ] Test edge cases (corrupted files, etc.)
- [ ] Manual testing on macOS/Linux

## 9. Risk Analysis

### Technical Risks

**State File Conflicts**
- Risk: User runs multiple instances simultaneously
- Likelihood: Low (typical usage is single CLI invocation)
- Impact: Low (last write wins, not critical)
- Mitigation: Document that concurrent usage not supported

**Performance Degradation**
- Risk: State lookup adds latency to searches
- Likelihood: Low (O(n) lookup on small arrays)
- Impact: Low (< 100 quests to check)
- Mitigation: Profile and optimize if needed (map instead of array)

**State File Corruption**
- Risk: JSON parsing fails on corrupted file
- Likelihood: Medium (power loss, disk errors)
- Impact: Medium (user loses completion data)
- Mitigation:
  - Graceful fallback to empty state
  - Atomic write with temp file + rename
  - Option to manually edit JSON if needed

**Backward Compatibility**
- Risk: Existing users expect old behavior
- Likelihood: High (any breaking changes)
- Impact: Medium (user confusion)
- Mitigation:
  - State is opt-in (no file = works as before)
  - --no-state flag preserves old behavior
  - Clear documentation of new feature

### User Experience Risks

**Accidental Completion Marking**
- Risk: User accidentally marks quest complete
- Likelihood: Medium (single spacebar press)
- Impact: Low (easily reversible by toggling again)
- Mitigation: No confirmation needed, toggle is instant

**Unclear State Impact**
- Risk: User doesn't understand why item safety changed
- Likelihood: Medium (implicit state effects)
- Impact: Medium (confusion about results)
- Mitigation: Show "Previously required by (completed)" messages

**State Sync Issues**
- Risk: User completes quest in-game but forgets to update tool
- Likelihood: High (manual tracking)
- Impact: Low (user still sees conservative recommendations)
- Mitigation: Document that state management is manual

## 10. Success Metrics

### Quantitative

**Performance**:
- State load time: < 10ms
- State save time: < 20ms
- Search with state filtering: < 5ms overhead
- State file size: < 10KB

**Adoption** (if telemetry added):
- % users creating state file: target 30%
- Average quests marked complete: target 10+
- State file retention: target 90% after 1 week

### Qualitative

**User Feedback**:
- "Finally stops telling me items are unsafe when I've done the quest"
- "Completion tracking makes the tool much more useful"
- "Easy to mark quests as I complete them"

**Developer Experience**:
- Clean separation of concerns (state module is isolated)
- Easy to extend (add new completion types)
- Well-tested (80%+ coverage on new code)

## 11. Future Enhancements (Post-MVP)

### v2 Features

**Auto-detection from save files**:
- Parse game save files to auto-populate completions
- Requires reverse engineering save format
- Eliminates manual tracking

**Cloud sync**:
- Optional account system for cross-device state
- GitHub gist integration?
- End-to-end encryption for privacy

**Completion stats**:
```bash
./arcitems --stats
# Shows:
# - 22 of 66 quests complete (33%)
# - 145 items now safe to recycle
# - Estimated value unlocked: 15,000 coins
```

**Quest dependency tree**:
- Show which quests unlock others
- Highlight "next recommended quest" for progression
- Visual tree in TUI

**Import/export state**:
```bash
./arcitems --export-state > my_progress.json
./arcitems --import-state < my_progress.json
```
Useful for backups or sharing with friends.

**Bulk completion**:
- Mark all quests from a trader as complete
- "Mark all Celeste quests complete"
- Useful for new users who've already progressed

**Completion timestamps**:
- Track when each quest was marked complete
- Show progression timeline
- Motivational stats ("You've completed 5 quests this week!")

## 12. Open Questions

### Technical Decisions

**Should state use arrays or maps for quest IDs?**
- Current: `[]string` (simpler, smaller JSON)
- Alternative: `map[string]bool` (faster lookup)
- Recommendation: Stick with arrays for MVP (< 100 quests, negligible perf difference)

**Should we validate questIDs against loaded data?**
- Option A: Silently ignore unknown questIDs (more resilient)
- Option B: Warn about unknown questIDs (helps catch errors)
- Recommendation: Option A - user might have old state, don't nag

**~~How to handle crafting table upgrades?~~**
- **RESOLVED**: Found hideout station data in `hideout/*.json`
- 6 stations with 3 upgrade levels each (weapon_bench, equipment_bench, etc.)
- State tracks completed level per station (0-3)
- workbench has maxLevel: 0 (no upgrades defined yet)

### Product Decisions

**Should completion manager be its own subcommand?**
```bash
# Option A: Flag-based (current spec)
./arcitems --manage

# Option B: Subcommand-based
./arcitems completion list
./arcitems completion toggle ss10h
./arcitems completion clear
```
Recommendation: Flag-based for MVP (simpler), subcommands for v2

**Should we show completion stats in search output?**
```bash
$ ./arcitems "battery"
Data version: 2025.11.21.2052 (448 items, 66 quests)
Completion: 22 quests complete (33%)
...
```
Recommendation: Yes, adds 1 line, useful context

**Should --no-state be needed or default to stateful?**
- Current: State used if present (opt-in via creating state file)
- Alternative: Require explicit --use-state flag
- Recommendation: Current approach (less friction)

## 13. Appendix: Example Workflows

### Workflow 1: New User

```bash
# Day 1: User installs tool
./arcitems "battery"
# Output: Battery ✗ (unsafe, required by quests)

# Day 5: User completes a few quests
./arcitems --manage
# (Marks 5 quests complete, saves)

# Day 5: Search again
./arcitems "battery"
# Output: Battery ✓ (safe, previously required by Quest X [completed])
```

### Workflow 2: Power User

```bash
# Bulk completion after progressing in game
./arcitems --manage
# (Marks 20 quests complete in one session)

# Search while gaming
./arcitems "broken flash" --json | jq '.safeToSell'
# Output: true

# Later: Review what's safe to sell
./arcitems --json "rare" | jq 'map(select(.safeToSell)) | .[].item.name.en'
# Output: List of safe rare items
```

### Workflow 3: Fresh Playthrough

```bash
# User wants to see all quest items (ignore old completions)
./arcitems "quest_item" --no-state
# Shows all quest requirements, regardless of completion state

# Useful for planning ahead or helping friends
```

## 14. Validation Checklist

**Spec Completeness**:
- [x] Technical architecture defined
- [x] Data structures specified
- [x] State persistence strategy documented
- [x] UI mockups provided
- [x] CLI integration planned
- [x] Testing strategy outlined
- [x] Risk analysis completed
- [x] Success metrics defined

**Implementation Ready**:
- [x] All new types defined in code snippets
- [x] File structure specified
- [x] Dependencies identified (no new deps needed)
- [x] Backward compatibility ensured
- [x] Performance targets set

**User Experience**:
- [x] Primary use case addressed (false positives eliminated)
- [x] Opt-in design (doesn't break existing workflows)
- [x] Clear visual indicators (✓ vs ✗, "Previously required by")
- [x] Simple interaction model (space to toggle, s to save)

This specification is ready for implementation.
