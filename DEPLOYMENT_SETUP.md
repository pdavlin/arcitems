# Deployment Setup Guide

This document outlines the steps needed to set up the automated deployment system for ArcItems.

## Prerequisites

1. GitHub repository created at `github.com/pdavlin/arcitems`
2. GitHub account with admin access
3. Homebrew tap repository will need to be created

## Step 1: Push Code to GitHub

Initialize and push the repository:

```bash
cd /Users/pdavlin/Development/arcitems
git init
git add .
git commit -m "Initial commit with automated deployment system"
git branch -M main
git remote add origin https://github.com/pdavlin/arcitems.git
git push -u origin main
```

## Step 2: Create Homebrew Tap Repository

Create a new GitHub repository at `github.com/pdavlin/homebrew-arcitems`:

```bash
mkdir -p ~/homebrew-arcitems
cd ~/homebrew-arcitems
git init
mkdir -p Formula
```

The formula will be automatically populated by the GitHub Actions workflow.

Create initial README:

```bash
cat > README.md <<EOF
# Homebrew Tap for ArcItems

Official Homebrew tap for [ArcItems](https://github.com/pdavlin/arcitems).

## Installation

\`\`\`bash
brew tap pdavlin/arcitems
brew install arcitems
\`\`\`

## Updates

This tap is automatically updated by GitHub Actions when new data is available.
EOF

git add .
git commit -m "Initial commit"
git branch -M main
git remote add origin https://github.com/pdavlin/homebrew-arcitems.git
git push -u origin main
```

## Step 3: Create GitHub Personal Access Token

Create a Personal Access Token with the following permissions:
- Repository access: `homebrew-arcitems` (or all repositories)
- Permissions:
  - `contents: write` (to push formula updates)
  - `metadata: read` (required)

Save this token - you'll need it for the next step.

## Step 4: Create Scoop Bucket Repository

Create a new GitHub repository at `github.com/pdavlin/scoop-arcitems`:

```bash
mkdir -p ~/scoop-arcitems
cd ~/scoop-arcitems
git init
mkdir -p .github/workflows bucket
```

Use the files from the `scoop-bucket-files/` directory in the main repository:

```bash
# Copy files from the main repository
cp -r /path/to/arcitems/scoop-bucket-files/* .

# Push to GitHub
git add .
git commit -m "Initial Scoop bucket setup"
git branch -M main
git remote add origin https://github.com/pdavlin/scoop-arcitems.git
git push -u origin main
```

Configure repository settings:
1. Go to Settings > Actions > General
2. Under "Actions permissions", select "Allow all actions and reusable workflows"
3. Under "Workflow permissions", select "Read and write permissions"
4. Click "Save"

## Step 5: Create GitHub Personal Access Tokens

Create two Personal Access Tokens:

### Token 1: Homebrew Tap Access

1. Go to Settings > Developer settings > Personal access tokens > Fine-grained tokens
2. Click "Generate new token"
3. Token name: `homebrew-arcitems-sync`
4. Repository access: Select "Only select repositories" > Choose `homebrew-arcitems`
5. Permissions:
   - `contents: write` (to push formula updates)
   - `metadata: read` (required)
6. Generate token and save it

### Token 2: Scoop Bucket Access

1. Click "Generate new token" again
2. Token name: `scoop-arcitems-sync`
3. Repository access: Select "Only select repositories" > Choose `scoop-arcitems`
4. Permissions:
   - `contents: write` (to push manifest updates)
   - `metadata: read` (required)
5. Generate token and save it

## Step 6: Add Repository Secrets

In the `arcitems` repository settings:

1. Go to Settings > Secrets and variables > Actions
2. Add the following secrets:
   - Name: `TAP_REPO_TOKEN`, Value: Token from Step 5 (Token 1)
   - Name: `SCOOP_BUCKET_TOKEN`, Value: Token from Step 5 (Token 2)

## Step 7: Test the Workflow

### Manual Trigger

1. Go to the Actions tab in the `arcitems` repository
2. Select "Sync Data and Release" workflow
3. Click "Run workflow" button
4. Select the `main` branch
5. Click "Run workflow"

This will:
- Fetch the latest data from upstream
- Build binaries for all platforms
- Create a GitHub release
- Update the Homebrew formula
- Update the Scoop manifest

### Verify the Release

1. Check the Releases page for a new release
2. Verify all binaries are attached
3. Check the `homebrew-arcitems` repository for the updated formula
4. Check the `scoop-arcitems` repository for the updated manifest

### Test Homebrew Installation

```bash
brew tap pdavlin/arcitems
brew install arcitems
arcitems --version
```

### Test Scoop Installation (Windows)

On a Windows machine with Scoop installed:

```powershell
# Add the bucket
scoop bucket add pdavlin https://github.com/pdavlin/scoop-arcitems

# Install arcitems
scoop install arcitems

# Verify installation
arcitems --version
arcitems "broken flash"

# Test update detection
scoop update
scoop status
```

## Step 8: Update Python Script Permissions

Ensure the fetch script has upstream SHA tracking:

The sync workflow already handles this by adding the `upstreamSHA` field to metadata.json.

## Automated Sync Schedule

The workflow runs automatically:
- Every 6 hours (00:00, 06:00, 12:00, 18:00 UTC)
- Only creates releases when upstream data changes
- Completely hands-off after initial setup

## Monitoring

Check workflow status:
- GitHub Actions tab shows all workflow runs
- Email notifications for failed workflows (configure in GitHub settings)
- Release feed: `https://github.com/pdavlin/arcitems/releases.atom`

## Troubleshooting

### Workflow fails to update Homebrew formula

Check:
1. `TAP_REPO_TOKEN` secret is set correctly
2. Token has `contents: write` permission
3. Tap repository exists and is accessible

### Binaries fail to build

Check:
1. Go version in workflow matches project requirements
2. All dependencies are available
3. Cross-compilation flags are correct

### No release created despite data changes

Check:
1. Data actually changed in upstream repository
2. Commit was successful
3. Version tag generation succeeded

Look at the workflow logs for detailed error messages.

## Update Checker Testing

Test the update checker locally:

```bash
# Build with a fake old version (>7 days old)
go build -ldflags "-X main.Version=2025.11.17.0000" -o arcitems ./cmd/arcitems

# Run the tool (should show update notification if newer version exists)
./arcitems "broken flash"

# Second run (should NOT show notification - already notified about this version)
./arcitems "broken flash"

# Check cache file
cat ~/.arcitems/update_check.json

# Test with config file to permanently disable
echo '{"disableUpdateCheck": true}' > ~/.arcitems/config.json
./arcitems "broken flash"

# Test with command-line flag to disable
./arcitems "broken flash" --no-update-check

# Test custom notification threshold
echo '{"notifyThresholdDays": 1}' > ~/.arcitems/config.json
go build -ldflags "-X main.Version=2025.11.23.0000" -o arcitems ./cmd/arcitems
./arcitems "broken flash"  # Should notify (2 days old)
```

## Maintenance

The system requires minimal maintenance:

1. Monitor workflow runs for failures
2. Rotate GitHub tokens annually
3. Update Go version in workflows as needed
4. Review upstream data changes periodically

## Next Steps

After initial setup:

1. Create an initial release manually or let the workflow create one
2. Test installation via Homebrew
3. Announce to users
4. Monitor for issues

## Support

For issues with:
- **Deployment system**: Check workflow logs and this guide
- **Homebrew installation**: Check tap repository and formula
- **Scoop installation**: Check bucket repository and manifest
- **Data sync**: Verify upstream repository is accessible
- **Update checker**: Check `~/.arcitems/update_check.json` cache file and `~/.arcitems/config.json` config file
