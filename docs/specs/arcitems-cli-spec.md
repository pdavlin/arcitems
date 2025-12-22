# ArcItems CLI Tool - Feature Specification

## Executive Summary

**Feature Name**: ArcItems CLI - ARC Raiders Item Query Tool

**Business Value**: Enables ARC Raiders players to quickly search items, determine quest usage, and make informed sell/recycle decisions without tabbing between game and web resources.

**Complexity Score**: 5/10 (Medium)
- 8-12 source files
- Multiple Go dependencies (Bubble Tea, fuzzy search)
- Embedded data strategy (offline-first)
- Automated data sync via GitHub Actions
- No database (JSON files)

**Estimated Effort**: 2-3 days for MVP
- Day 1: Data embedding, parsing, quest analysis
- Day 2: Bubble Tea UI, fuzzy search integration
- Day 3: Polish, testing, documentation, Homebrew setup

**Language**: Go (due to Bubble Tea framework requirement)

## 1. Dataset Analysis

### Available Data Sources

From https://github.com/RaidTheory/arcraiders-data:

**Item Data** (items/*.json):
- 200+ individual item files
- Fields: id, name, rarity, type, value, recyclesInto, salvagesInto, foundIn, weightKg, stackSize
- No native quest-usage indicators

**Quest Data** (quests/*.json):
- 86 quest files
- Fields: id, name, trader, requiredItemIds[], rewardItemIds[]
- Item requirements: `{"itemId": "item_id", "quantity": 1}`

**Project Data** (projects.json):
- Multi-phase projects (Expedition)
- Phase requirements: requirementItemIds[], requirementCategories[]

**Critical Insight**: Items don't know they're used in quests. The tool must build reverse mapping by scanning all quests and projects for item references.

### Data Relationships

```
Quest -> requiredItemIds[] -> Item
Project -> requirementItemIds[] -> Item
Project -> requirementCategories[] -> Item (by category match)
```

## 2. Technical Architecture

### High-Level Design (Offline-First)

```
┌─────────────────────────────────────────────┐
│              CLI Entry Point                │
│   Parse args: `arcitems broken flash`      │
│   Check for updates (once/day, optional)    │
└──────────────────┬──────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────┐
│         Embedded Data Module                │
│  - Data bundled in binary (go:embed)        │
│  - No network calls during search           │
│  - Items, quests, projects all embedded     │
└──────────────────┬──────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────┐
│         Quest Analyzer Module               │
│  - Build item -> quest mapping              │
│  - Build item -> project mapping            │
│  - Compute usage index                      │
└──────────────────┬──────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────┐
│         Fuzzy Search Module                 │
│  - Use github.com/lithammer/fuzzysearch     │
│  - Search item names (multilingual?)        │
│  - Rank by Levenshtein distance             │
└──────────────────┬──────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────┐
│       Bubble Tea UI Module                  │
│  - Display search results                   │
│  - Show quest usage per item                │
│  - Indicate safe to sell/recycle            │
│  - Navigation: arrow keys, enter            │
└─────────────────────────────────────────────┘

External Update Flow (GitHub Actions):
┌─────────────────────────────────────────────┐
│     GitHub Actions (runs every 6 hours)     │
│  - Monitor RaidTheory/arcraiders-data       │
│  - Detect changes to upstream repo          │
│  - Bundle new data into binary              │
│  - Create new release with binaries         │
│  - Update Homebrew formula                  │
└─────────────────────────────────────────────┘
```

### File Structure

```
arcitems/
├── .github/
│   └── workflows/
│       └── sync-data.yml         # Auto-sync upstream data
├── cmd/
│   └── arcitems/
│       └── main.go               # CLI entry point
├── internal/
│   ├── data/
│   │   ├── bundled/             # Embedded data directory
│   │   │   ├── items.json       # All items (embedded)
│   │   │   ├── quests.json      # All quests (embedded)
│   │   │   ├── projects.json    # Projects (embedded)
│   │   │   └── metadata.json    # Data version info
│   │   ├── embedded.go          # go:embed declarations
│   │   └── types.go             # Item, Quest, Project structs
│   ├── analyzer/
│   │   ├── quest_analyzer.go   # Build quest -> item mappings
│   │   └── usage.go             # Determine if item is safe to sell
│   ├── search/
│   │   └── fuzzy.go             # Fuzzy search implementation
│   ├── ui/
│   │   ├── model.go             # Bubble Tea model
│   │   ├── view.go              # UI rendering
│   │   └── update.go            # Event handling
│   └── update/
│       └── check.go             # Update notification system
├── homebrew/
│   └── arcitems.rb              # Homebrew formula
├── go.mod
├── go.sum
├── README.md
└── docs/
    └── specs/
        └── arcitems-cli-spec.md # This file
```

### Data Types (Go)

```go
// internal/data/types.go

type Item struct {
    ID           string            `json:"id"`
    Name         map[string]string `json:"name"` // multilingual
    Rarity       string            `json:"rarity"`
    Type         string            `json:"type"`
    Value        int               `json:"value"`
    RecyclesInto []Material        `json:"recyclesInto"`
    SalvagesInto []Material        `json:"salvagesInto"`
    FoundIn      string            `json:"foundIn"`
    WeightKg     float64           `json:"weightKg"`
    StackSize    int               `json:"stackSize"`
}

type Material struct {
    ItemID   string `json:"itemId"`
    Quantity int    `json:"quantity"`
}

type Quest struct {
    ID              string         `json:"id"`
    Name            map[string]string `json:"name"`
    RequiredItemIds []ItemRequirement `json:"requiredItemIds"`
    RewardItemIds   []ItemRequirement `json:"rewardItemIds"`
}

type ItemRequirement struct {
    ItemID   string `json:"itemId"`
    Quantity int    `json:"quantity"`
}

type Project struct {
    Phases []Phase `json:"phases"`
}

type Phase struct {
    RequirementItemIds   []ItemRequirement `json:"requirementItemIds"`
    RequirementCategories []CategoryReq    `json:"requirementCategories"`
}

type CategoryReq struct {
    Category      string `json:"category"`
    ValueRequired int    `json:"valueRequired"`
}

type ItemUsage struct {
    Item          *Item
    UsedInQuests  []string // quest IDs
    UsedInProjects []string // project IDs
    SafeToSell    bool
}
```

## 3. Core Functionality

### Command-Line Interface

```bash
# Basic usage
arcitems broken flash

# Display results in interactive mode (default)
# Showing: 5 matches for "broken flash"
# ┌─────────────────────────────────────────┐
# │ ● Broken Flashlight                      │
# │   Rare | Recyclable | 1000 coins         │
# │   ✓ Safe to sell/recycle                 │
# │   ─────────────────────────────────────  │
# │   Not required for any quests            │
# │   Recycles into: 2x Battery, 6x Metal    │
# └─────────────────────────────────────────┘
```

**Optional Flags**:
```bash
arcitems broken flash --json           # JSON output
arcitems broken flash --lang en        # Language (default: en)
arcitems broken flash --no-update-check # Disable update notification
arcitems --check-update                # Check for new data version
arcitems --version                     # Show version and data info
```

### Quest Analysis Logic

```go
// Pseudocode for determining if an item is safe to sell

func (a *Analyzer) IsItemSafeToSell(itemID string) bool {
    // Check all quests
    for _, quest := range a.quests {
        for _, req := range quest.RequiredItemIds {
            if req.ItemID == itemID {
                return false // Used in quest
            }
        }
    }

    // Check all projects
    for _, project := range a.projects {
        for _, phase := range project.Phases {
            for _, req := range phase.RequirementItemIds {
                if req.ItemID == itemID {
                    return false // Used in project
                }
            }
            // Check category requirements
            item := a.items[itemID]
            for _, catReq := range phase.RequirementCategories {
                if item.Type == catReq.Category {
                    return false // May be needed for category requirement
                }
            }
        }
    }

    return true // Safe to sell
}
```

### Fuzzy Search Implementation

```go
// internal/search/fuzzy.go

import "github.com/lithammer/fuzzysearch/fuzzy"

func SearchItems(query string, items map[string]*Item) []SearchResult {
    results := []SearchResult{}

    for id, item := range items {
        // Search English name by default
        name := item.Name["en"]
        if fuzzy.Match(query, name) {
            distance := levenshtein(query, name)
            results = append(results, SearchResult{
                Item:     item,
                Distance: distance,
            })
        }
    }

    // Sort by Levenshtein distance (lower = better match)
    sort.Slice(results, func(i, j int) bool {
        return results[i].Distance < results[j].Distance
    })

    return results
}
```

### Data Embedding Strategy (Offline-First)

**Embedded Data Location**: `internal/data/bundled/`

**Embedded Files** (via go:embed):
- `items.json` - All item data
- `quests.json` - All quest data
- `projects.json` - Project data
- `metadata.json` - Data version and sync timestamp

**Data Update Flow**:
1. GitHub Actions monitors RaidTheory/arcraiders-data every 6 hours
2. On upstream changes: Actions bundles new data, commits to repo
3. Actions creates new release with version tag (e.g., v2025.11.21.1430)
4. Actions builds binaries for all platforms with embedded data
5. Actions updates Homebrew formula with new version
6. Users update via package manager: `brew upgrade arcitems`

**Update Notification**:
- CLI checks GitHub releases API once per 24 hours (non-blocking, < 100ms)
- Cache file: `~/.arcitems/update_check.json` (only stores last check time)
- Displays friendly message: "💡 New data available: v2025.11.23 (you have v2025.11.21)"
- Shows appropriate update command based on installation method
- Can be disabled with `--no-update-check` flag

**Performance Target**: < 50ms for all queries (100% offline, no network calls)

## 4. Distribution & Update Mechanism

### Package Manager Distribution

**Primary: Homebrew (macOS/Linux)**
```bash
# Installation
brew tap pdavlin/arcitems
brew install arcitems

# Updates
brew upgrade arcitems
```

**Secondary: Direct Downloads (All platforms)**
- GitHub releases page provides binaries for:
  - macOS (arm64, amd64)
  - Linux (amd64, arm64)
  - Windows (amd64)

### Automated Data Sync (GitHub Actions)

**Workflow: `.github/workflows/sync-data.yml`**

Runs every 6 hours and:
1. Clones RaidTheory/arcraiders-data repository
2. Compares latest commit SHA with last synced version
3. If changed:
   - Aggregates individual JSON files into bundled format
   - Commits new data to `internal/data/bundled/`
   - Creates version tag (format: `vYYYY.MM.DD.HHMM`)
   - Builds cross-platform binaries with embedded data
   - Creates GitHub release with all binaries
   - Updates Homebrew formula with new SHA256

**No manual intervention required** - data stays current automatically.

### Update Check Implementation

**File: `internal/update/check.go`**

```go
func NotifyIfOutdated(currentVersion string, disableCheck bool) {
    // Check once per 24 hours
    if shouldCheckForUpdates() {
        latestVersion := fetchLatestVersion() // GitHub releases API
        if latestVersion != currentVersion {
            updateCmd := detectUpdateCommand() // "brew upgrade arcitems"
            fmt.Fprintf(os.Stderr, "💡 New data available: %s\n", latestVersion)
            fmt.Fprintf(os.Stderr, "   Update: %s\n", updateCmd)
        }
    }
}
```

**Package Manager Detection**:
- Checks executable path for "Cellar" or "homebrew" → `brew upgrade arcitems`
- Checks for "scoop" → `scoop update arcitems`
- Checks for "/usr/bin" on Linux → `apt upgrade arcitems`
- Fallback → "download from GitHub releases"

**Cache File**: `~/.arcitems/update_check.json`
- Only stores: `{"lastCheck": "2025-11-21T14:30:00Z"}`
- No data storage, minimal disk usage

**User Experience**:
```bash
$ arcitems broken flash
💡 New data available: v2025.11.23 (you have v2025.11.21)
   Update: brew upgrade arcitems

Search: broken flash
● Broken Flashlight ✓
...
```

User runs update once, notification disappears on next run.

### Offline Behavior

**Works completely offline**:
- All searches use embedded data
- Update check fails silently if offline
- No error messages about network
- Tool never blocks waiting for network

**Update check can be disabled**:
```bash
arcitems broken flash --no-update-check  # Skip update check
```

Useful for:
- CI/CD scripts
- Fully offline environments
- Users who don't want notifications

### Homebrew Formula Details

**File: `homebrew/arcitems.rb`**

```ruby
class Arcitems < Formula
  desc "CLI tool for ARC Raiders item and quest lookup"
  homepage "https://github.com/pdavlin/arcitems"
  url "https://github.com/pdavlin/arcitems/archive/refs/tags/v2025.11.21.tar.gz"
  sha256 "..." # Auto-updated by GitHub Actions
  version "2025.11.21"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w -X main.Version=#{version}")
  end
end
```

GitHub Actions updates this file automatically on each release.

## 5. Bubble Tea UI Implementation

### Model Structure

```go
type model struct {
    searchQuery  string
    results      []ItemUsage
    cursor       int
    viewport     viewport.Model
    err          error
}

func (m model) Init() tea.Cmd {
    return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "q", "esc", "ctrl+c":
            return m, tea.Quit
        case "up", "k":
            if m.cursor > 0 {
                m.cursor--
            }
        case "down", "j":
            if m.cursor < len(m.results)-1 {
                m.cursor++
            }
        }
    }
    return m, nil
}

func (m model) View() string {
    var b strings.Builder

    b.WriteString(fmt.Sprintf("Search: %s\n\n", m.searchQuery))

    for i, result := range m.results {
        cursor := " "
        if i == m.cursor {
            cursor = "●"
        }

        safeIcon := "✗"
        if result.SafeToSell {
            safeIcon = "✓"
        }

        b.WriteString(fmt.Sprintf("%s %s %s\n", cursor, result.Item.Name["en"], safeIcon))

        if i == m.cursor {
            // Show details for selected item
            b.WriteString(fmt.Sprintf("  %s | %s | %d coins\n",
                result.Item.Rarity, result.Item.Type, result.Item.Value))

            if len(result.UsedInQuests) > 0 {
                b.WriteString(fmt.Sprintf("  Required by quests: %v\n", result.UsedInQuests))
            } else {
                b.WriteString("  Not required for any quests\n")
            }

            if len(result.Item.RecyclesInto) > 0 {
                b.WriteString("  Recycles into: ")
                for _, mat := range result.Item.RecyclesInto {
                    b.WriteString(fmt.Sprintf("%dx %s ", mat.Quantity, mat.ItemID))
                }
                b.WriteString("\n")
            }
        }
        b.WriteString("\n")
    }

    return b.String()
}
```

## 6. Implementation Checklist

### Pre-Implementation
- [x] Analyze ArcRaiders data structure
- [x] Research Bubble Tea framework
- [ ] Set up Go module and project structure
- [ ] Install dependencies (Bubble Tea, fuzzysearch)
- [ ] Create internal/data/bundled/ directory
- [ ] Set up GitHub Actions workflow
- [ ] Create Homebrew formula

### Core Implementation - Phase 1: Data Layer
- [ ] Fetch initial data snapshot from arcraiders-data
- [ ] Create bundled/*.json files in repo
- [ ] Define Go structs for Item, Quest, Project
- [ ] Implement go:embed for data files
- [ ] Add JSON unmarshaling with error handling
- [ ] Write unit tests for data loading

### Core Implementation - Phase 2: Analysis Engine
- [ ] Build quest -> item reverse mapping
- [ ] Implement project -> item mapping
- [ ] Create IsItemSafeToSell() logic
- [ ] Handle category-based requirements
- [ ] Cache computed usage index
- [ ] Write unit tests for analyzer

### Core Implementation - Phase 3: Search
- [ ] Integrate lithammer/fuzzysearch library
- [ ] Implement Levenshtein distance ranking
- [ ] Support multilingual search (stretch goal)
- [ ] Optimize search performance for 200+ items
- [ ] Write search benchmarks

### Core Implementation - Phase 4: UI
- [ ] Create Bubble Tea model/view/update
- [ ] Implement keyboard navigation (up/down/q)
- [ ] Design item detail view layout
- [ ] Add color coding (safe=green, unsafe=red)
- [ ] Handle empty search results gracefully
- [ ] Test UI responsiveness

### Core Implementation - Phase 5: Distribution & Updates
- [ ] Implement update checker (internal/update/check.go)
- [ ] Add package manager detection logic
- [ ] Create GitHub Actions sync workflow
- [ ] Set up Homebrew tap repository
- [ ] Configure automated releases

### Integration
- [ ] Connect all modules via main.go
- [ ] Add command-line flag parsing (--check-update, --json, --no-update-check)
- [ ] Implement JSON output mode
- [ ] Add version flag with data info
- [ ] Handle errors gracefully (parsing, UI)

### Validation
- [ ] Unit tests (target: 80% coverage)
  - Data loader edge cases
  - Quest analyzer correctness
  - Fuzzy search ranking accuracy
- [ ] Integration tests
  - End-to-end search flow
  - Cache invalidation
  - Data update workflow
- [ ] Manual testing
  - Test with known quest items (should show unsafe)
  - Test with junk items (should show safe)
  - Test with partial/typo queries
  - Verify performance on slow connections

### Documentation
- [ ] README with installation instructions
- [ ] Usage examples and screenshots
- [ ] Contribution guidelines
- [ ] License file (match arcraiders-data license)

## 7. Risk Analysis

### Technical Risks

**Breaking Changes from Upstream Data**
- Risk: RaidTheory changes JSON schema
- Likelihood: Medium (game in active development)
- Impact: High (tool stops working)
- Mitigation:
  - Version cached data with schema version
  - Add schema validation on data load
  - Fail gracefully with clear error messages
  - Monitor upstream repo for changes

**Performance Issues**
- Risk: 200+ items, 86 quests = slow analysis
- Likelihood: Low-Medium
- Impact: Medium (poor UX if > 500ms)
- Mitigation:
  - Pre-compute usage index, cache it
  - Lazy load quest details only when needed
  - Profile with pprof if issues arise

**Network Dependency**
- Risk: GitHub API unavailable for update checks
- Likelihood: Low (GitHub uptime is high)
- Impact: Low (only affects update notifications, not functionality)
- Mitigation:
  - Data embedded in binary, works 100% offline
  - Update check fails silently when offline
  - Only queries GitHub releases API once per day
  - Can be disabled with --no-update-check flag
  - Tool remains fully functional without network

**Multilingual Support Complexity**
- Risk: 18 languages per item, unclear which to search
- Likelihood: High
- Impact: Low (nice-to-have feature)
- Mitigation:
  - Start with English only (MVP)
  - Add --lang flag for language selection
  - Defer full multilingual fuzzy search to v2

**Category Requirement Ambiguity**
- Risk: Project phase 5 uses categories, not specific items
- Likelihood: Medium (affects accuracy)
- Impact: Medium (false negatives on "safe to sell")
- Mitigation:
  - Mark items in required categories as "potentially needed"
  - Add warning: "May be needed for category requirement"
  - Allow user override with --ignore-categories flag

### Security Risks

**Supply Chain Attacks**
- Risk: Malicious dependency or compromised upstream data
- Likelihood: Low
- Impact: High
- Mitigation:
  - Pin dependency versions in go.mod
  - Use go.sum for checksum verification
  - Only fetch from official GitHub repository
  - No arbitrary code execution from data files

**Data Integrity**
- Risk: Embedded data corrupted or tampered with
- Likelihood: Very Low (binary compiled from source)
- Impact: High (incorrect quest info)
- Mitigation:
  - Data embedded at compile time, immutable
  - Install via trusted package managers (Homebrew)
  - GitHub Actions signs releases
  - Users can rebuild from source to verify

## 8. Dependencies

### External Dependencies

```go
// go.mod
module github.com/pdavlin/arcitems

go 1.21

require (
    github.com/charmbracelet/bubbletea v0.25.0
    github.com/charmbracelet/lipgloss v0.9.1  // Styling
    github.com/lithammer/fuzzysearch v1.1.8   // Fuzzy search
    github.com/spf13/cobra v1.8.0             // CLI framework
)
```

**Rationale**:
- **Bubble Tea**: Requirement for interactive TUI
- **Lipgloss**: Styling companion to Bubble Tea
- **fuzzysearch**: Lightweight, pure Go, no deps
- **cobra**: Industry-standard CLI tool framework

### Internal Dependencies

All internal modules depend on `internal/data/types.go`:
- `internal/data/loader.go` (fetches data)
- `internal/analyzer/quest_analyzer.go` (consumes items/quests)
- `internal/search/fuzzy.go` (consumes items)
- `internal/ui/model.go` (displays results)

**Dependency Flow**:
```
main.go
  → update.NotifyIfOutdated() [optional, non-blocking]
  → data.LoadEmbedded()
  → analyzer.BuildUsageIndex()
  → search.SearchItems()
  → ui.RunBubbleTea()
```

## 9. Testing Strategy

### Unit Tests

```go
// internal/analyzer/quest_analyzer_test.go

func TestIsItemSafeToSell_QuestItem(t *testing.T) {
    analyzer := &Analyzer{
        quests: []Quest{
            {
                ID: "broken_monument",
                RequiredItemIds: []ItemRequirement{
                    {ItemID: "first_wave_tape", Quantity: 1},
                },
            },
        },
    }

    if analyzer.IsItemSafeToSell("first_wave_tape") {
        t.Error("Quest item should not be safe to sell")
    }
}

func TestIsItemSafeToSell_JunkItem(t *testing.T) {
    analyzer := &Analyzer{
        quests: []Quest{},
        projects: []Project{},
    }

    if !analyzer.IsItemSafeToSell("broken_flashlight") {
        t.Error("Non-quest item should be safe to sell")
    }
}
```

### Integration Tests

```go
// internal/integration_test.go

func TestEndToEndSearch(t *testing.T) {
    // Setup: Load test data fixture
    loader := data.NewLoader("testdata/")
    items, quests, projects := loader.LoadAll()

    analyzer := analyzer.NewAnalyzer(items, quests, projects)
    searcher := search.NewSearcher(items, analyzer)

    // Execute: Search for "broken flash"
    results := searcher.Search("broken flash")

    // Assert: Should find "broken_flashlight"
    if len(results) == 0 {
        t.Fatal("Expected at least 1 result")
    }

    if results[0].Item.ID != "broken_flashlight" {
        t.Errorf("Expected broken_flashlight, got %s", results[0].Item.ID)
    }

    if !results[0].SafeToSell {
        t.Error("Broken flashlight should be safe to sell")
    }
}
```

### Manual Testing Checklist

**Basic Functionality**:
- [ ] Search with exact item name returns correct item
- [ ] Search with typo ("flaslight") still finds item
- [ ] Quest-required item shows "✗ Not safe to sell"
- [ ] Non-quest item shows "✓ Safe to sell/recycle"
- [ ] Multiple results show in correct ranked order

**Edge Cases**:
- [ ] Empty search query shows helpful message
- [ ] No matches found displays gracefully
- [ ] Item with no recycle materials displays correctly
- [ ] Item required by multiple quests lists all quests

**Performance**:
- [ ] All searches complete in < 50ms (offline)
- [ ] Binary size < 10MB with embedded data
- [ ] UI remains responsive with 20+ results
- [ ] Update check adds < 100ms on first daily run when online

**Error Handling**:
- [ ] Works completely offline (no network dependency)
- [ ] Update check fails silently when offline
- [ ] Invalid embedded JSON shows parsing error details
- [ ] Missing item ID in quest doesn't crash

**Usability**:
- [ ] Keyboard navigation (arrows, j/k) works smoothly
- [ ] Quit commands (q, esc, ctrl+c) exit immediately
- [ ] Color coding distinguishes safe vs unsafe items
- [ ] Selected item shows full details

## 10. Success Metrics

### Quantitative Metrics

**Performance**:
- Search latency: < 50ms (P95) - fully offline
- Binary size: < 10MB (with embedded data)
- Memory footprint: < 50MB
- Update check latency: < 100ms when online (once per day)

**Accuracy**:
- Quest item detection: 100% (no false negatives)
- Non-quest item detection: > 95% (few false positives OK)
- Fuzzy search relevance: Top result correct 90% of time

**Adoption** (if open-sourced):
- GitHub stars: 50+ in first month
- Weekly active users: 100+ (if telemetry added)

### Qualitative Metrics

**User Satisfaction**:
- "Saves me from checking wiki" - primary use case
- "Fast enough to use mid-game" - performance goal
- "Accurate quest item warnings" - trust factor

**Developer Experience**:
- Clear error messages when data fetch fails
- Easy to understand codebase for contributors
- Comprehensive README with examples

**Community Engagement** (stretch goal):
- 5+ external contributors
- Active issue tracker for data corrections
- Integration with other ARC Raiders tools

## 11. Rollout Plan

### Phase 1: MVP (Week 1)
**Goal**: Functional CLI with basic search

- [ ] Fetch initial data snapshot from arcraiders-data
- [ ] Embed data using go:embed
- [ ] Build quest analyzer (items vs quests only)
- [ ] Integrate fuzzy search
- [ ] Create basic Bubble Tea UI
- [ ] Test with 10-20 items manually

**Success Criteria**:
- Can search for items with fuzzy matching (offline)
- Shows if item is used in quests
- Runs on macOS/Linux/Windows
- Binary size < 10MB

### Phase 2: Polish (Week 2)
**Goal**: Production-ready tool

- [ ] Add project/category requirement analysis
- [ ] Improve UI with colors and better layout
- [ ] Add command-line flags (--json, --check-update, --no-update-check)
- [ ] Implement update notification system
- [ ] Write comprehensive tests
- [ ] Create installation documentation

**Success Criteria**:
- 80% test coverage
- All known quest items detected
- Professional UI experience
- Update checks work without breaking offline usage

### Phase 3: Distribution (Week 3)
**Goal**: Public release with automated updates

- [ ] Set up GitHub Actions sync workflow
- [ ] Create Homebrew formula and tap
- [ ] Configure automated releases on data changes
- [ ] Set up cross-platform binary builds
- [ ] Write detailed README with screenshots
- [ ] Post to ARC Raiders community (Reddit, Discord)
- [ ] Monitor feedback and bug reports

**Success Criteria**:
- Homebrew installation works smoothly
- GitHub Actions successfully detects and syncs data changes
- Automated releases include all platforms
- 50+ stars on GitHub
- Zero critical bugs reported
- Positive community feedback

## 12. Future Enhancements (Post-MVP)

**v2 Features**:
- Interactive mode: Start without args, type to search live
- Multilingual fuzzy search (search in any language)
- Item comparison mode: "Compare broken_flashlight vs broken_lamp"
- Trader price integration from trades.json
- Export results to CSV/JSON file
- Web UI version (compile to WASM?)

**Advanced Analytics**:
- "Most valuable junk items" report
- Quest item checklist mode
- Optimal recycle path calculations
- Skill node integration (items needed for upgrades)

**Community Features**:
- Crowdsourced notes on items ("This is rare, don't sell!")
- Integration with ARC Raiders companion apps
- Discord bot version

## 13. Research Links

**Go Libraries**:
- Bubble Tea: https://github.com/charmbracelet/bubbletea
- Bubbletea examples: https://github.com/charmbracelet/bubbletea/tree/master/examples
- Fuzzysearch: https://github.com/lithammer/fuzzysearch
- Cobra CLI: https://github.com/spf13/cobra

**Data Source**:
- ArcRaiders Data: https://github.com/RaidTheory/arcraiders-data
- Data structure examples: https://github.com/RaidTheory/arcraiders-data/blob/main/items/broken_flashlight.json

**Fuzzy Search Algorithms**:
- Levenshtein distance: https://en.wikipedia.org/wiki/Levenshtein_distance
- Fuzzy search best practices: https://www.meilisearch.com/blog/fuzzy-search

**CLI Design Patterns**:
- Go CLI tutorial: https://spf13.com/presentation/building-an-awesome-cli-app-in-go-oscon/
- TUI design principles: https://charm.sh/blog/tui-design/

## 14. Open Questions

**Technical Decisions**:
1. ~~Should we fetch data from GitHub API or clone the repo?~~
   - **RESOLVED**: Embed data in binary using go:embed, update via GitHub Actions

2. How to handle multilingual item names in search?
   - **Recommendation**: Default to English, add --lang flag for v2

3. Should we pre-compute usage index or calculate on-demand?
   - **Recommendation**: Pre-compute and cache (86 quests is manageable)

4. JSON output mode: structured or pretty-printed?
   - **Recommendation**: Structured for scripting, add --pretty flag

**Product Decisions**:
1. Should "category requirement" items be marked unsafe?
   - **Recommendation**: Yes, with explanation (better safe than sorry)

2. ~~Do we need authentication for higher GitHub API limits?~~
   - **RESOLVED**: No, only check releases API once per day for notifications

3. Should we show item images in terminal (with sixel/kitty protocol)?
   - **Recommendation**: Defer to v2 (complex, limited terminal support)

4. How should users update to get new data?
   - **RESOLVED**: Via package manager (brew upgrade arcitems), GitHub Actions handles releases

## 15. Appendix: Example Output

### Scenario 1: Quest Item (Unsafe to Sell)

```
$ arcitems first wave tape

Search: first wave tape

● First Wave Tape ✗
  Rare | Quest Item | 500 coins
  ⚠ Required by quests:
    - Broken Monument (Tian Wen)
  Recycles into: 3x Paper, 1x Plastic

  Advice: Do not sell or recycle until quest is complete.

Press q to quit, arrows to navigate
```

### Scenario 2: Junk Item (Safe to Sell)

```
$ arcitems broken flash

Search: broken flash

● Broken Flashlight ✓
  Rare | Recyclable | 1000 coins
  ✓ Safe to sell/recycle
  Not required for any quests or projects
  Recycles into: 2x Battery, 6x Metal Parts

  Advice: Recycle for crafting materials or sell for coins.

  Broken Lamp
  Common | Recyclable | 300 coins
  ✓ Safe to sell/recycle

Press q to quit, arrows to navigate
```

### Scenario 3: JSON Output Mode

```
$ arcitems broken flash --json

[
  {
    "item": {
      "id": "broken_flashlight",
      "name": "Broken Flashlight",
      "rarity": "Rare",
      "type": "Recyclable",
      "value": 1000,
      "recyclesInto": [
        {"itemId": "battery", "quantity": 2},
        {"itemId": "metal_parts", "quantity": 6}
      ]
    },
    "usedInQuests": [],
    "usedInProjects": [],
    "safeToSell": true
  }
]
```

## 16. Validation Checklist

**Spec Completeness**:
- [x] Technical architecture defined
- [x] Data model specified
- [x] Implementation steps outlined
- [x] Testing strategy documented
- [x] Risk analysis completed
- [x] Success metrics defined
- [x] Rollout plan created

**Feasibility**:
- [x] All data sources accessible (GitHub public repo)
- [x] Required libraries available and stable
- [x] No blocking technical dependencies
- [x] Estimated effort realistic (2-3 days)

**Clarity**:
- [x] User stories clear and testable
- [x] Technical specifications actionable
- [x] Example outputs provided
- [x] Open questions documented

This specification is ready for implementation. 🚀
