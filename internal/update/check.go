package update

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pdavlin/arcitems/internal/config"
)

const (
	githubReleasesAPI = "https://api.github.com/repos/pdavlin/arcitems/releases/latest"
	checkInterval   = 24 * time.Hour
	notifyThreshold = 0 * time.Hour // Notify immediately on any new version (configurable via notifyThresholdDays)
)

type cacheData struct {
	LastCheck       time.Time `json:"lastCheck"`
	LastSeenVersion string    `json:"lastSeenVersion"` // Track last version shown to avoid duplicate notifications
}

type releaseInfo struct {
	TagName string `json:"tag_name"`
}

// NotifyIfOutdated checks for updates and displays a notification if a newer version is available.
// It respects the 24-hour check interval and fails silently if offline or if any errors occur.
func NotifyIfOutdated(currentVersion string, disableCheck bool) {
	// Check config file for disable setting
	cfg, err := config.Load()
	if err == nil && cfg.DisableUpdateCheck {
		return
	}

	if disableCheck {
		return
	}

	// Skip if we're running the dev version
	if currentVersion == "dev" {
		return
	}

	// Check if we should perform an update check
	if !shouldCheckForUpdates(cfg) {
		return
	}

	// Fetch latest version (with timeout)
	latestVersion, err := fetchLatestVersion()
	if err != nil {
		// Fail silently - user experience is not affected
		return
	}

	// Load cache to check last seen version
	cache, err := loadCache()
	if err != nil {
		cache = &cacheData{}
	}

	// Update the last check time and seen version
	if err := updateLastCheckTime(latestVersion); err != nil {
		// Non-critical, continue anyway
	}

	// Compare versions - only notify if version is old enough and we haven't notified about this version
	if latestVersion != "" && latestVersion != currentVersion && shouldNotifyForVersion(currentVersion, latestVersion, cache.LastSeenVersion, cfg) {
		updateCmd := detectUpdateCommand()
		fmt.Fprintf(os.Stderr, "💡 New data available: %s (you have %s)\n", latestVersion, currentVersion)
		fmt.Fprintf(os.Stderr, "   Update: %s\n\n", updateCmd)
	}
}

// shouldCheckForUpdates returns true if we haven't checked in the configured interval
func shouldCheckForUpdates(cfg *config.Config) bool {
	cache, err := loadCache()
	if err != nil {
		// If we can't load the cache, allow the check
		return true
	}

	interval := checkInterval
	if cfg != nil && cfg.UpdateCheckInterval > 0 {
		interval = time.Duration(cfg.UpdateCheckInterval) * time.Hour
	}

	return time.Since(cache.LastCheck) >= interval
}

// shouldNotifyForVersion determines if we should notify about a new version
func shouldNotifyForVersion(currentVersion, latestVersion, lastSeenVersion string, cfg *config.Config) bool {
	// Already notified about this version
	if latestVersion == lastSeenVersion {
		return false
	}

	// Parse version timestamps (format: YYYY.MM.DD.HHMM)
	currentTime, err := parseVersionTimestamp(currentVersion)
	if err != nil {
		// If we can't parse versions, fall back to simple comparison
		return true
	}

	latestTime, err := parseVersionTimestamp(latestVersion)
	if err != nil {
		// If we can't parse versions, fall back to simple comparison
		return true
	}

	// Determine threshold from config or use default
	threshold := notifyThreshold
	if cfg != nil && cfg.NotifyThresholdDays > 0 {
		threshold = time.Duration(cfg.NotifyThresholdDays) * 24 * time.Hour
	}

	// Only notify if version is older than threshold
	age := latestTime.Sub(currentTime)
	return age > threshold
}

// parseVersionTimestamp parses version string in format YYYY.MM.DD.HHMM
func parseVersionTimestamp(version string) (time.Time, error) {
	parts := strings.Split(version, ".")
	if len(parts) != 4 {
		return time.Time{}, fmt.Errorf("invalid version format")
	}

	year, err := strconv.Atoi(parts[0])
	if err != nil {
		return time.Time{}, err
	}

	month, err := strconv.Atoi(parts[1])
	if err != nil {
		return time.Time{}, err
	}

	day, err := strconv.Atoi(parts[2])
	if err != nil {
		return time.Time{}, err
	}

	hourMin := parts[3]
	if len(hourMin) != 4 {
		return time.Time{}, fmt.Errorf("invalid time format")
	}

	hour, err := strconv.Atoi(hourMin[:2])
	if err != nil {
		return time.Time{}, err
	}

	min, err := strconv.Atoi(hourMin[2:])
	if err != nil {
		return time.Time{}, err
	}

	return time.Date(year, time.Month(month), day, hour, min, 0, 0, time.UTC), nil
}

// fetchLatestVersion fetches the latest release tag from GitHub API
func fetchLatestVersion() (string, error) {
	client := &http.Client{
		Timeout: 2 * time.Second, // Quick timeout to avoid blocking
	}

	req, err := http.NewRequest("GET", githubReleasesAPI, nil)
	if err != nil {
		return "", err
	}

	// Set user agent (GitHub API requires it)
	req.Header.Set("User-Agent", "arcitems-cli")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var release releaseInfo
	if err := json.Unmarshal(body, &release); err != nil {
		return "", err
	}

	// Strip 'v' prefix if present
	version := strings.TrimPrefix(release.TagName, "v")
	return version, nil
}

// detectUpdateCommand detects how the user installed the tool and returns the appropriate update command
func detectUpdateCommand() string {
	execPath, err := os.Executable()
	if err != nil {
		return "download from https://github.com/pdavlin/arcitems/releases"
	}

	// Check for Scoop (Windows user-scoped install)
	if strings.Contains(execPath, "\\scoop\\") || strings.Contains(execPath, "/scoop/") {
		return "scoop update arcitems"
	}

	// Check for Homebrew installation
	if strings.Contains(execPath, "/Cellar/") || strings.Contains(execPath, "/homebrew/") {
		return "brew upgrade arcitems"
	}

	// Check for Linuxbrew
	if strings.Contains(execPath, "/.linuxbrew/") {
		return "brew upgrade arcitems"
	}

	// Check for system installation on Linux
	if strings.HasPrefix(execPath, "/usr/local/bin") || strings.HasPrefix(execPath, "/usr/bin") {
		return "download from https://github.com/pdavlin/arcitems/releases"
	}

	// Default fallback
	return "download from https://github.com/pdavlin/arcitems/releases"
}

// getCacheFilePath returns the path to the update check cache file
func getCacheFilePath() (string, error) {
	configDir, err := config.GetConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, "update_check.json"), nil
}

// loadCache loads the update check cache from disk
func loadCache() (*cacheData, error) {
	cachePath, err := getCacheFilePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Cache doesn't exist yet, return empty cache
			return &cacheData{}, nil
		}
		return nil, err
	}

	var cache cacheData
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}

	return &cache, nil
}

// updateLastCheckTime updates the last check time and seen version in the cache
func updateLastCheckTime(latestVersion string) error {
	cachePath, err := getCacheFilePath()
	if err != nil {
		return err
	}

	cache := cacheData{
		LastCheck:       time.Now(),
		LastSeenVersion: latestVersion,
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cachePath, data, 0644)
}
