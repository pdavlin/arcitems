# ArcItems - ARC Raiders Item Query Tool

Command-line tool for searching ARC Raiders items and determining whether they are safe to sell or recycle based on quest requirements.

## Features

- **Fuzzy search** - Find items even with typos or partial matches
- **Quest detection** - Instantly know if an item is required for quests
- **Project tracking** - Check if items are needed for expedition projects
- **Hideout tracking** - See which items are needed for hideout station upgrades
- **Completion tracking** - Mark quests and hideout levels as complete to get accurate recyclability recommendations
- **Offline-first** - All data is embedded in the binary, no network required
- **Interactive UI** - Browse search results with arrow keys
- **JSON output** - Machine-readable format for scripting
- **Multilingual** - Search in 18 languages

## Installation

### Homebrew (macOS/Linux)

```bash
brew tap pdavlin/arcitems
brew install arcitems
```

To update:
```bash
brew upgrade arcitems
```

### Scoop (Windows)

```powershell
scoop bucket add pdavlin https://github.com/pdavlin/scoop-arcitems
scoop install arcitems
```

To update:
```powershell
scoop update arcitems
```

### Direct Download

Download pre-built binaries for your platform from the [releases page](https://github.com/pdavlin/arcitems/releases).

Available platforms:
- macOS (Apple Silicon and Intel)
- Linux (amd64 and arm64)
- Windows (amd64)

### From Source

```bash
git clone https://github.com/pdavlin/arcitems.git
cd arcitems
go build -o arcitems ./cmd/arcitems
./arcitems "broken flash"
```

## Usage

### Basic search

```bash
arcitems broken flash
```

This will launch an interactive UI showing:
- Item name with rarity color
- Safe to sell indicator (✓ or ✗)
- Quest usage information
- Recycle/salvage materials
- Item value and properties

### Completion Manager

Track your progress to get accurate recommendations:

```bash
arcitems --manage
# or
arcitems -m
```

In the completion manager:
- `Space` - Toggle quest/project completion
- `+`/`-` or `→`/`←` - Adjust hideout station level
- `↑`/`↓` or `j`/`k` - Navigate
- `Tab` - Switch between sections
- `s` - Save and exit
- `q` - Quit without saving

Your completion state is saved to `~/.arcitems/completion.json` and automatically applied to future searches.

### Navigation

- `↑`/`↓` or `j`/`k` - Move between results
- `q` or `Esc` - Quit

### JSON output

```bash
arcitems broken flash --json
```

Output results in JSON format for scripting or integration with other tools.

### Language selection

```bash
arcitems "lampe torche" --lang fr
```

Search item names in different languages (default: English).

### Ignore completion state

```bash
arcitems battery --no-state
```

Search without applying your completion state (shows all quest/hideout requirements regardless of completion).

## Examples

### Find a safe-to-sell item

```bash
$ arcitems broken flash
Data version: 2025.11.21.2052 (448 items, 66 quests)

Search: broken flash

Found 1 match(es)

● Broken Flashlight ✓
  Rare | Recyclable | 1000 coins
  ✓ Safe to sell/recycle
  Recycles into: 2x battery, 6x metal_parts

↑/k: up | ↓/j: down | q: quit
```

### Check a quest item

```bash
$ arcitems "first wave tape" --json
[
  {
    "item": {
      "id": "first_wave_tape",
      "name": {
        "en": "First Wave Tape"
      },
      ...
    },
    "usedInQuests": [
      "ss10h"
    ],
    "usedInProjects": null,
    "safeToSell": false
  }
]
```

## Configuration

### Update Notifications

The tool automatically checks for data updates once per 24 hours (when online). By default, you'll only see notifications if your version is more than 7 days old.

To customize update behavior, create `~/.arcitems/config.json`:

```json
{
  "disableUpdateCheck": false,
  "updateCheckInterval": 24,
  "notifyThresholdDays": 7
}
```

**Options**:
- `disableUpdateCheck`: Permanently disable update notifications (default: false)
- `updateCheckInterval`: Hours between update checks (default: 24)
- `notifyThresholdDays`: Minimum age difference to show notification (default: 7)

You can also use the `--no-update-check` flag for one-time disabling:
```bash
arcitems "broken flash" --no-update-check
```

When an update is available, you'll see a notification with the appropriate command for your installation method:

```
💡 New data available: 2025.11.24.1628 (you have 2025.11.17.1200)
   Update: brew upgrade arcitems
```

## Data Updates

Item and quest data is sourced from [RaidTheory/arcraiders-data](https://github.com/RaidTheory/arcraiders-data) and embedded directly in the binary.

### Data Sync Process

Data is automatically synced every 6 hours via GitHub Actions:
1. Monitor upstream repository for changes
2. Fetch and bundle new data
3. Build cross-platform binaries
4. Create GitHub release
5. Update Homebrew formula and Scoop manifest

No manual intervention required after initial setup.

### Manual Data Update (Development)

For development purposes, you can manually update the data:

```bash
python3 scripts/fetch_data.py
go build -o arcitems ./cmd/arcitems
```

## Development

### Project structure

```
arcitems/
├── cmd/arcitems/        # CLI entry point
├── internal/
│   ├── analyzer/        # Quest, project, and hideout analysis
│   ├── data/            # Data types and embedded JSON
│   ├── search/          # Fuzzy search implementation
│   ├── state/           # Completion state management
│   └── ui/              # Bubble Tea UI (search + completion manager)
└── scripts/             # Data fetching utilities
```

### Building

```bash
go build -o arcitems ./cmd/arcitems
```

### Testing

```bash
# Test search functionality
./arcitems "broken flash" --json

# Test quest detection
./arcitems "first wave tape" --json

# Test interactive UI
./arcitems battery
```

## License

MIT

## Credits

- Game data from [RaidTheory/arcraiders-data](https://github.com/RaidTheory/arcraiders-data)
- Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) TUI framework
- Fuzzy search powered by [fuzzysearch](https://github.com/lithammer/fuzzysearch)
