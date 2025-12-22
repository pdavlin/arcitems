# Windows Packaging & Update Notification Enhancement Specification

## Executive Summary

**Feature Name**: Windows Distribution via Scoop + Enhanced Update Notifications

**Business Value**:
- Expand Windows user base (no Homebrew equivalent currently)
- Improve update discovery without being intrusive
- Maintain package manager best practices (no self-updating)

**Complexity Score**: 4/10 (Medium)
- 3-5 files to modify
- New Scoop bucket repository needed
- GitHub Actions workflow additions
- No database changes, no new dependencies

**Estimated Effort**: 4-6 hours
- 2 hours: Scoop bucket setup and testing
- 2 hours: Update notification improvements
- 1-2 hours: Documentation and validation

## Background: Package Manager Best Practices

### Homebrew Official Stance

From Homebrew documentation:
> "Software that can upgrade itself does not integrate well with Homebrew formulae's own upgrade functionality. The self-update functionality should be disabled."

**Key principles**:
- CLI tools (formulae) should NEVER self-update
- Self-updating breaks package manager version tracking
- Only GUI applications (casks) can self-update (marked with `auto_updates true`)
- Update notifications are acceptable if non-intrusive

### Current Implementation

Our current approach:
- ✅ No self-updating (respects package manager)
- ✅ Update notifications (informs users)
- ❌ Checks every 24 hours regardless of version age
- ❌ Can be repetitive for users on old versions
- ❌ No Windows package manager support

## Technical Architecture

### 1. Update Notification Enhancements

#### Current Behavior

```go
// internal/update/check.go
const checkInterval = 24 * time.Hour

type cacheData struct {
    LastCheck time.Time `json:"lastCheck"`
}
```

Problems:
- Checks every 24 hours since last check, not since version release
- Shows notification even if user's version is only 1 day old
- Shows same notification repeatedly if user doesn't update
- No way to permanently disable without flag

#### Proposed Changes

```go
// internal/update/check.go modifications

type cacheData struct {
    LastCheck       time.Time `json:"lastCheck"`
    LastSeenVersion string    `json:"lastSeenVersion"`  // NEW: Track last version shown
}

const (
    checkInterval     = 24 * time.Hour
    notifyThreshold   = 7 * 24 * time.Hour  // NEW: Only notify if >7 days old
)
```

**Enhanced notification logic**:
1. Check if current version is >7 days older than latest release
2. Only show notification once per new version (track `LastSeenVersion`)
3. Add config file support for permanent disabling
4. Detect Scoop installation and show appropriate update command

#### New Config File

Location: `~/.arcitems/config.json`

```json
{
  "disableUpdateCheck": false,
  "updateCheckInterval": 24,
  "notifyThresholdDays": 7
}
```

Options:
- `disableUpdateCheck`: Permanently disable (alternative to `--no-update-check` flag)
- `updateCheckInterval`: Hours between checks (default: 24)
- `notifyThresholdDays`: Minimum age difference to show notification (default: 7)

### 2. Scoop Bucket Setup

#### What is Scoop?

Scoop is a command-line package manager for Windows:
- No admin rights required (installs to user directory)
- Most similar to Homebrew in philosophy
- Developer-focused, portable installations
- JSON-based manifest format
- Auto-update support via "Excavator" GitHub Actions

#### Repository Structure

```
scoop-arcitems/
├── .github/
│   └── workflows/
│       ├── excavator.yml        # Auto-update manifests
│       └── issue-handling.yml   # Auto-close issues
├── bucket/
│   └── arcitems.json            # Main manifest
└── README.md
```

#### Manifest Format

File: `bucket/arcitems.json`

```json
{
  "version": "2025.11.24.1628",
  "description": "CLI tool for ARC Raiders item and quest lookup",
  "homepage": "https://github.com/pdavlin/arcitems",
  "license": "MIT",
  "architecture": {
    "64bit": {
      "url": "https://github.com/pdavlin/arcitems/releases/download/v2025.11.24.1628/arcitems-windows-amd64.zip",
      "hash": "sha256:...",
      "bin": "arcitems-windows-amd64.exe",
      "shortcuts": [
        ["arcitems-windows-amd64.exe", "ArcItems"]
      ]
    }
  },
  "checkver": {
    "github": "https://github.com/pdavlin/arcitems"
  },
  "autoupdate": {
    "architecture": {
      "64bit": {
        "url": "https://github.com/pdavlin/arcitems/releases/download/v$version/arcitems-windows-amd64.zip",
        "hash": {
          "url": "https://github.com/pdavlin/arcitems/releases/download/v$version/checksums.txt",
          "regex": "^([a-f0-9]{64})\\s+arcitems-windows-amd64\\.zip$"
        }
      }
    }
  }
}
```

Key components:
- `checkver`: Enables automatic version detection from GitHub
- `autoupdate`: Template for updating URLs and extracting checksums
- `hash`: Security verification using SHA256 from checksums.txt

#### Excavator Workflow

File: `scoop-arcitems/.github/workflows/excavator.yml`

```yaml
name: Excavator
on:
  schedule:
    - cron: '0 */6 * * *'  # Every 6 hours
  workflow_dispatch:

jobs:
  excavate:
    runs-on: windows-latest
    steps:
      - uses: actions/checkout@v4

      - name: Excavator
        uses: ScoopInstaller/GithubActions@main
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          SKIP_UPDATED: '1'
```

This workflow:
- Runs every 6 hours automatically
- Checks for new releases on GitHub
- Uses `autoupdate` section to generate new manifest
- Commits updated manifest to repository
- No manual intervention required

### 3. GitHub Actions Integration

#### Update Main Sync Workflow

Add to `.github/workflows/sync-data.yml` after Homebrew formula update:

```yaml
- name: Update Scoop manifest
  if: steps.check.outputs.needs_sync == 'true' && steps.changes.outputs.has_changes == 'true'
  env:
    SCOOP_BUCKET_TOKEN: ${{ secrets.SCOOP_BUCKET_TOKEN }}
  run: |
    VERSION="${{ steps.version.outputs.version }}"
    TAG="${{ steps.version.outputs.tag }}"
    WIN_SHA256=$(grep "arcitems-windows-amd64.zip" dist/checksums.txt | awk '{print $1}')

    # Clone scoop bucket
    git clone https://x-access-token:${SCOOP_BUCKET_TOKEN}@github.com/pdavlin/scoop-arcitems.git scoop-repo
    cd scoop-repo

    # Update manifest
    jq --arg version "$VERSION" \
       --arg url "https://github.com/pdavlin/arcitems/releases/download/$TAG/arcitems-windows-amd64.zip" \
       --arg hash "sha256:$WIN_SHA256" \
       '.version = $version | .architecture."64bit".url = $url | .architecture."64bit".hash = $hash' \
       bucket/arcitems.json > temp.json && mv temp.json bucket/arcitems.json

    # Commit and push
    git config user.name "github-actions[bot]"
    git config user.email "github-actions[bot]@users.noreply.github.com"
    git add bucket/arcitems.json
    git commit -m "Update arcitems to $VERSION"
    git push
```

This ensures the Scoop manifest is updated simultaneously with:
- Homebrew formula
- GitHub release creation
- Binary builds

## Implementation Checklist

### Phase 1: Update Notification Improvements (2 hours)

**Code Changes**:
- [ ] Add `LastSeenVersion` field to `cacheData` struct
- [ ] Add `notifyThreshold` constant (7 days)
- [ ] Implement version age comparison logic
- [ ] Add config file loading (`~/.arcitems/config.json`)
- [ ] Update `detectUpdateCommand()` to detect Scoop
- [ ] Only show notification once per version
- [ ] Write unit tests for new logic

**Files Modified**:
- `internal/update/check.go` - Enhanced notification logic
- `internal/update/check_test.go` - New test cases
- `README.md` - Document config file options

**Version Comparison Logic**:
```go
func shouldNotifyForVersion(currentVersion, latestVersion, lastSeenVersion string) bool {
    // Already notified about this version
    if latestVersion == lastSeenVersion {
        return false
    }

    // Parse version timestamps (format: YYYY.MM.DD.HHMM)
    currentTime, _ := parseVersionTimestamp(currentVersion)
    latestTime, _ := parseVersionTimestamp(latestVersion)

    // Only notify if version is >7 days newer
    age := latestTime.Sub(currentTime)
    return age > notifyThreshold
}

func parseVersionTimestamp(version string) (time.Time, error) {
    // Parse "2025.11.24.1628" format
    parts := strings.Split(version, ".")
    if len(parts) != 4 {
        return time.Time{}, fmt.Errorf("invalid version format")
    }

    year, _ := strconv.Atoi(parts[0])
    month, _ := strconv.Atoi(parts[1])
    day, _ := strconv.Atoi(parts[2])
    hourMin := parts[3]
    hour, _ := strconv.Atoi(hourMin[:2])
    min, _ := strconv.Atoi(hourMin[2:])

    return time.Date(year, time.Month(month), day, hour, min, 0, 0, time.UTC), nil
}
```

**Scoop Detection**:
```go
func detectUpdateCommand() string {
    execPath, err := os.Executable()
    if err != nil {
        return "download from https://github.com/pdavlin/arcitems/releases"
    }

    // Check for Scoop (user-scoped install)
    if strings.Contains(execPath, "\\scoop\\") || strings.Contains(execPath, "/scoop/") {
        return "scoop update arcitems"
    }

    // Check for Homebrew
    if strings.Contains(execPath, "/Cellar/") || strings.Contains(execPath, "/homebrew/") {
        return "brew upgrade arcitems"
    }

    // ... rest of detection logic
}
```

### Phase 2: Scoop Bucket Repository (2 hours)

**Repository Setup**:
- [ ] Create `scoop-arcitems` repository on GitHub
- [ ] Enable GitHub Actions (Settings > Actions > Allow all actions)
- [ ] Set workflow permissions (Settings > Actions > Read and write permissions)
- [ ] Add bucket manifest (`bucket/arcitems.json`)
- [ ] Configure Excavator workflow
- [ ] Add README with installation instructions
- [ ] Test manual installation

**New Repository**: `github.com/pdavlin/scoop-arcitems`

**README.md Template**:
```markdown
# Scoop Bucket for ArcItems

Official Scoop bucket for [ArcItems](https://github.com/pdavlin/arcitems).

## Installation

\`\`\`powershell
scoop bucket add pdavlin https://github.com/pdavlin/scoop-arcitems
scoop install arcitems
\`\`\`

## Updates

This bucket is automatically updated by Excavator when new releases are published.

To update:
\`\`\`powershell
scoop update arcitems
\`\`\`

## About

ArcItems is a CLI tool for ARC Raiders item and quest lookup. See the [main repository](https://github.com/pdavlin/arcitems) for more information.
```

### Phase 3: GitHub Actions Integration (1 hour)

**Workflow Changes**:
- [ ] Add Scoop manifest update step to `sync-data.yml`
- [ ] Create Personal Access Token for scoop-arcitems repository
- [ ] Add `SCOOP_BUCKET_TOKEN` secret to main repository
- [ ] Test end-to-end release process
- [ ] Verify Scoop Excavator detects updates

**Files Modified**:
- `.github/workflows/sync-data.yml` - Add Scoop update step

**Testing**:
```bash
# Trigger manual workflow run
gh workflow run sync-data.yml

# Verify Scoop manifest was updated
curl https://raw.githubusercontent.com/pdavlin/scoop-arcitems/main/bucket/arcitems.json | jq .version
```

### Phase 4: Documentation & Testing (1 hour)

**Documentation**:
- [ ] Update README.md with Scoop installation section
- [ ] Update DEPLOYMENT_SETUP.md with Scoop setup instructions
- [ ] Add config file documentation
- [ ] Add troubleshooting section for Windows

**Testing Checklist**:
- [ ] Build test binary with old version
- [ ] Verify notification threshold (7 days)
- [ ] Test notification suppression (same version)
- [ ] Test config file override
- [ ] Install via Scoop: `scoop bucket add pdavlin https://github.com/pdavlin/scoop-arcitems && scoop install arcitems`
- [ ] Verify Scoop update detection: `scoop status`
- [ ] Test binary execution on Windows 10 and 11

## Risk Analysis

### Technical Risks

**Scoop Bucket Maintenance**
- **Risk**: Excavator fails to auto-update manifests
- **Likelihood**: Low (official GitHub Actions, well-tested)
- **Impact**: Medium (manual updates needed)
- **Mitigation**:
  - Manual update workflow as fallback
  - Monitor GitHub Actions notifications
  - Test autoupdate before relying on it

**Update Notification Spam**
- **Risk**: Users still find notifications annoying
- **Likelihood**: Low (7-day threshold reduces frequency)
- **Impact**: Low (can be disabled)
- **Mitigation**:
  - Config file for permanent disable
  - Only show once per version
  - Clear message about how to disable

**Windows Binary Compatibility**
- **Risk**: Binary doesn't work on older Windows versions
- **Likelihood**: Low (Go has good Windows support)
- **Impact**: Medium (excludes some users)
- **Mitigation**:
  - Test on Windows 10 and 11
  - Document minimum requirements
  - Static linking ensures no DLL dependencies

**Version Parsing Fragility**
- **Risk**: Version format changes break comparison logic
- **Likelihood**: Low (format is established)
- **Impact**: High (notifications break)
- **Mitigation**:
  - Comprehensive unit tests
  - Fallback to simple string comparison
  - Document version format requirements

### User Experience Risks

**Discovery**
- **Risk**: Windows users don't find Scoop installation method
- **Likelihood**: Medium (Scoop less known than Chocolatey)
- **Impact**: Medium (miss potential users)
- **Mitigation**:
  - Prominently document in README
  - Add to GitHub release notes
  - Provide direct download as alternative

**Confusion with Multiple Package Managers**
- **Risk**: Users install via multiple methods
- **Likelihood**: Low
- **Impact**: Low (update detection handles both)
- **Mitigation**:
  - Document recommended method per platform
  - Update detection works for all methods
  - Clear uninstall instructions

**First-Time Windows Setup**
- **Risk**: Users unfamiliar with Scoop struggle with installation
- **Likelihood**: Medium
- **Impact**: Low (direct download available)
- **Mitigation**:
  - Step-by-step Scoop installation guide
  - Link to official Scoop documentation
  - Provide direct download as easier alternative

## Success Metrics

### Quantitative
- Scoop installation works on first try (manual test)
- Update notifications shown at most once per week per version
- Zero false positive update notifications
- Windows binary downloads increase by 20%+ after Scoop release
- Config file correctly disables notifications

### Qualitative
- Users report easy Windows installation via Scoop
- No complaints about notification spam
- Package manager correctly detected in update messages
- Positive feedback on config file flexibility

## Testing Strategy

### Update Notification Testing

**Version Age Tests**:
```bash
# Test notification for old version (>7 days)
go build -ldflags "-X main.Version=2025.11.17.0000" -o arcitems-test ./cmd/arcitems
./arcitems-test "battery"  # Should show notification

# Test no notification for recent version (<7 days)
go build -ldflags "-X main.Version=2025.11.23.0000" -o arcitems-test ./cmd/arcitems
./arcitems-test "battery"  # Should NOT show notification
```

**Notification Suppression Tests**:
```bash
# Test notification shown once per version
./arcitems-test "battery"  # First run - shows notification
./arcitems-test "battery"  # Second run - should NOT show

# Verify cache file
cat ~/.arcitems/update_check.json
# Should show: {"lastCheck": "...", "lastSeenVersion": "2025.11.24.1628"}
```

**Config File Tests**:
```bash
# Test permanent disable
echo '{"disableUpdateCheck": true}' > ~/.arcitems/config.json
./arcitems-test "battery"  # Should not check or notify

# Test custom threshold
echo '{"notifyThresholdDays": 1}' > ~/.arcitems/config.json
go build -ldflags "-X main.Version=2025.11.23.0000" -o arcitems-test ./cmd/arcitems
./arcitems-test "battery"  # Should show notification (2 days old)
```

**Package Manager Detection Tests**:
```bash
# Test Scoop detection (mock path)
# Requires modifying test to inject path
# Should output: "scoop update arcitems"

# Test Homebrew detection
# Should output: "brew upgrade arcitems"

# Test fallback
# Should output: "download from https://github.com/pdavlin/arcitems/releases"
```

### Scoop Installation Testing

**Initial Installation**:
```powershell
# Install Scoop (if not already installed)
irm get.scoop.sh | iex

# Add bucket
scoop bucket add pdavlin https://github.com/pdavlin/scoop-arcitems

# Install arcitems
scoop install arcitems

# Verify installation
arcitems --version
arcitems "broken flash"
```

**Update Testing**:
```powershell
# Check for updates
scoop update

# Check status (should show if outdated)
scoop status

# Update arcitems
scoop update arcitems

# Verify new version
arcitems --version
```

**Uninstall Testing**:
```powershell
# Uninstall
scoop uninstall arcitems

# Verify removal
where.exe arcitems  # Should not find
```

### Cross-Platform Testing Matrix

| Platform | Package Manager | Test Status |
|----------|----------------|-------------|
| macOS (arm64) | Homebrew | ✅ Tested |
| macOS (amd64) | Homebrew | ✅ Tested |
| Linux (amd64) | Direct download | ✅ Tested |
| Linux (arm64) | Direct download | ⏳ To test |
| Windows 11 | Scoop | ⏳ To test |
| Windows 10 | Scoop | ⏳ To test |
| Windows 11 | Direct download | ⏳ To test |

## Rollout Plan

### Phase 1: Update Notifications (Week 1, Days 1-2)

**Day 1: Implementation**
- Implement notification improvements
- Add config file support
- Write unit tests
- Update documentation

**Day 2: Testing & Deployment**
- Test with dev builds
- Verify notification threshold works
- Test config file override
- Merge to main (behind feature flag if desired)

**Success Criteria**:
- All tests passing
- No notifications for versions <7 days old
- Config file successfully disables notifications

### Phase 2: Scoop Bucket (Week 1, Days 3-4)

**Day 3: Bucket Setup**
- Create scoop-arcitems repository
- Add manifest with autoupdate
- Configure Excavator workflow
- Write installation documentation

**Day 4: Integration & Testing**
- Add Scoop update to sync-data workflow
- Test manual trigger of workflow
- Verify manifest updates correctly
- Test installation on Windows

**Success Criteria**:
- Scoop installation works
- Excavator detects new releases
- Manifest auto-updates correctly

### Phase 3: Documentation & Launch (Week 1, Day 5)

**Day 5: Finalization**
- Update all documentation
- Add Windows section to README
- Update DEPLOYMENT_SETUP guide
- Create announcement post

**Launch Checklist**:
- [ ] Scoop bucket is public and accessible
- [ ] README prominently features Scoop installation
- [ ] All workflows tested end-to-end
- [ ] Update notifications tested on all platforms
- [ ] Config file documented with examples

### Phase 4: Monitoring (Week 2+)

**Week 2: Active Monitoring**
- Monitor GitHub Actions for failures
- Watch for user feedback on notifications
- Check Scoop installation reports
- Address any immediate issues

**Ongoing**:
- Monthly review of notification frequency complaints
- Quarterly review of Windows installation metrics
- Update documentation based on user questions

## Documentation Updates

### README.md Changes

**Installation Section**:
```markdown
## Installation

### Homebrew (macOS/Linux)

\`\`\`bash
brew tap pdavlin/arcitems
brew install arcitems
\`\`\`

To update:
\`\`\`bash
brew upgrade arcitems
\`\`\`

### Scoop (Windows)

\`\`\`powershell
scoop bucket add pdavlin https://github.com/pdavlin/scoop-arcitems
scoop install arcitems
\`\`\`

To update:
\`\`\`powershell
scoop update arcitems
\`\`\`

### Direct Download

Download pre-built binaries from the [releases page](https://github.com/pdavlin/arcitems/releases).

Available platforms:
- macOS (Apple Silicon and Intel)
- Linux (amd64 and arm64)
- Windows (amd64)

### From Source

\`\`\`bash
git clone https://github.com/pdavlin/arcitems.git
cd arcitems
go build -o arcitems ./cmd/arcitems
\`\`\`
```

**Configuration Section**:
```markdown
## Configuration

### Update Notifications

The tool checks for data updates once per 24 hours. By default, you'll only see notifications if your version is more than 7 days old.

To customize update behavior, create \`~/.arcitems/config.json\`:

\`\`\`json
{
  "disableUpdateCheck": false,
  "updateCheckInterval": 24,
  "notifyThresholdDays": 7
}
\`\`\`

**Options**:
- \`disableUpdateCheck\`: Permanently disable update notifications (default: false)
- \`updateCheckInterval\`: Hours between update checks (default: 24)
- \`notifyThresholdDays\`: Minimum age difference to show notification (default: 7)

You can also use the \`--no-update-check\` flag for one-time disabling:
\`\`\`bash
arcitems "broken flash" --no-update-check
\`\`\`
```

### DEPLOYMENT_SETUP.md Additions

**Scoop Bucket Setup Section**:
```markdown
## Step 4: Create Scoop Bucket Repository

Create a new GitHub repository at \`github.com/pdavlin/scoop-arcitems\`:

\`\`\`bash
mkdir -p ~/scoop-arcitems
cd ~/scoop-arcitems
git init
mkdir -p .github/workflows bucket
\`\`\`

Create the manifest template in \`bucket/arcitems.json\` (see spec for full template).

Create the Excavator workflow in \`.github/workflows/excavator.yml\`.

Configure repository settings:
1. Settings > Actions > General > Actions permissions: "Allow all actions"
2. Settings > Actions > General > Workflow permissions: "Read and write permissions"

Push to GitHub:
\`\`\`bash
git add .
git commit -m "Initial Scoop bucket"
git branch -M main
git remote add origin https://github.com/pdavlin/scoop-arcitems.git
git push -u origin main
\`\`\`

## Step 5: Add Scoop Token to Main Repository

Create a Personal Access Token for updating the Scoop bucket:
1. Settings > Developer settings > Personal access tokens > Fine-grained tokens
2. Repository access: \`scoop-arcitems\`
3. Permissions: \`contents: write\`

Add to main repository secrets:
1. \`arcitems\` repository > Settings > Secrets and variables > Actions
2. New repository secret: \`SCOOP_BUCKET_TOKEN\`

## Step 6: Test Scoop Installation

\`\`\`powershell
# Install Scoop (if needed)
irm get.scoop.sh | iex

# Add bucket
scoop bucket add pdavlin https://github.com/pdavlin/scoop-arcitems

# Install
scoop install arcitems

# Verify
arcitems --version
\`\`\`
```

## Implementation Notes

### Version Format Dependency

The notification threshold logic depends on the version format: `YYYY.MM.DD.HHMM`

This format is already established in the project and is set by:
- `scripts/fetch_data.py`: `datetime.utcnow().strftime("%Y.%m.%d.%H%M")`
- `.github/workflows/sync-data.yml`: `VERSION=$(date -u +"%Y.%m.%d.%H%M")`

Any change to this format would require updating the `parseVersionTimestamp()` function.

### Config File Precedence

Priority order for update check settings:
1. Command-line flag (`--no-update-check`): Highest priority, disables for single run
2. Config file (`disableUpdateCheck`): Persistent across runs
3. Cache file (`lastSeenVersion`): Prevents duplicate notifications
4. Default behavior: Check every 24 hours, notify if >7 days old

### Scoop vs Chocolatey Trade-offs

We chose Scoop over Chocolatey because:

**Scoop Advantages**:
- No admin rights required (installs to user directory)
- More similar to Homebrew philosophy
- Developer-focused
- Better for CLI tools
- Free and open-source (no paywalled features)

**Chocolatey Disadvantages**:
- Requires admin rights for many packages
- Enterprise-focused (many features paywalled)
- More bureaucratic submission process
- Overkill for a simple CLI tool

**Winget Considerations**:
- Built into Windows 11 (best reach)
- But requires submitting to microsoft/winget-pkgs (more formal process)
- Can be added later as Phase 4 if Scoop adoption is strong

### Excavator Behavior

Excavator (Scoop's auto-updater) will:
- Run every 6 hours via cron schedule
- Check GitHub releases API for new versions
- Use `autoupdate` section to generate new manifest
- Calculate SHA256 from checksums.txt
- Commit changes to bucket repository
- Create pull request if configured (we auto-merge)

If Excavator fails:
- Main sync workflow also updates manifest (backup)
- Manual update is always possible via PR
- Users can install via direct download as fallback

## Open Questions

### Resolved
1. ~~Should we use Scoop, Chocolatey, or Winget for Windows?~~
   - **Decision**: Scoop (developer-focused, no admin required, most like Homebrew)

2. ~~Should update notifications respect package managers?~~
   - **Decision**: Yes, no self-updating, only notifications with appropriate commands

3. ~~How often should update checks happen?~~
   - **Decision**: Check every 24 hours, but only notify if version is >7 days old

### Pending
1. Should we support Winget in addition to Scoop?
   - **Recommendation**: Add as Phase 4 if demand exists
   - **Reason**: More reach but more process to submit

2. Should config file support be added to other settings?
   - **Recommendation**: Defer to future, focus on update notifications first
   - **Potential**: Completion state, language preference, display settings

3. Should we provide a Chocolatey package?
   - **Recommendation**: No, Scoop covers developer use case better
   - **Reason**: Chocolatey is enterprise-focused with paywalled features

## Appendix: Scoop Ecosystem

### Scoop Community Resources

**Official**:
- Main repository: https://github.com/ScoopInstaller/Scoop
- GitHub Actions: https://github.com/ScoopInstaller/GithubActions
- Bucket template: https://github.com/ScoopInstaller/BucketTemplate

**Popular Third-Party Buckets**:
- Extras: https://github.com/ScoopInstaller/Extras
- Java: https://github.com/ScoopInstaller/Java
- Games: https://github.com/Calinou/scoop-games

### Manifest Best Practices

From Scoop documentation:
- Always include `checkver` and `autoupdate` for automatic updates
- Use SHA256 hashes from checksums file when available
- Extract_dir should match the directory created by the archive
- Bin can be a string (single binary) or array (multiple binaries)
- Test installation on clean Windows VM before publishing

### Common Pitfalls

1. **Hash mismatch**: Ensure checksums.txt is correctly formatted
2. **Binary not found**: Check extract_dir matches actual directory in archive
3. **Excavator fails**: Verify autoupdate URL template matches actual release URLs
4. **Admin required**: Scoop should never need admin for installation

## Validation Checklist

Before marking this spec as complete:

- [x] Executive summary provides clear business value
- [x] Technical architecture is detailed and actionable
- [x] Implementation checklist covers all necessary steps
- [x] Risk analysis identifies potential issues and mitigations
- [x] Testing strategy is comprehensive
- [x] Rollout plan includes phases and success criteria
- [x] Documentation changes are specified
- [x] Code examples are provided for complex logic
- [x] Open questions are documented
- [x] Integration with existing systems is considered

This specification is ready for implementation.
