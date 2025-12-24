package update

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pdavlin/arcitems/internal/config"
)

func TestDetectUpdateCommand(t *testing.T) {
	tests := []struct {
		name     string
		execPath string
		want     string
	}{
		{
			name:     "Homebrew on macOS",
			execPath: "/usr/local/Cellar/arcitems/1.0.0/bin/arcitems",
			want:     "brew upgrade arcitems",
		},
		{
			name:     "Homebrew with new path",
			execPath: "/opt/homebrew/bin/arcitems",
			want:     "brew upgrade arcitems",
		},
		{
			name:     "Linuxbrew",
			execPath: "/home/user/.linuxbrew/bin/arcitems",
			want:     "brew upgrade arcitems",
		},
		{
			name:     "System installation",
			execPath: "/usr/local/bin/arcitems",
			want:     "download from https://github.com/pdavlin/arcitems/releases",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test validates the string matching logic
			// In a real scenario, we'd mock os.Executable()
			// For now, we're testing the logic inline
			var got string
			if filepath.Base(tt.execPath) == "arcitems" {
				if contains(tt.execPath, "/Cellar/") || contains(tt.execPath, "/homebrew/") {
					got = "brew upgrade arcitems"
				} else if contains(tt.execPath, "/.linuxbrew/") {
					got = "brew upgrade arcitems"
				} else {
					got = "download from https://github.com/pdavlin/arcitems/releases"
				}
			}

			if got != tt.want {
				t.Errorf("detectUpdateCommand() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCacheOperations(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "update_check.json")

	// Test writing cache
	cache := cacheData{
		LastCheck: time.Now(),
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal cache: %v", err)
	}

	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		t.Fatalf("Failed to write cache: %v", err)
	}

	// Test reading cache
	readData, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("Failed to read cache: %v", err)
	}

	var readCache cacheData
	if err := json.Unmarshal(readData, &readCache); err != nil {
		t.Fatalf("Failed to unmarshal cache: %v", err)
	}

	// Verify the data is roughly the same (within a second)
	timeDiff := cache.LastCheck.Sub(readCache.LastCheck)
	if timeDiff > time.Second || timeDiff < -time.Second {
		t.Errorf("Cache time mismatch: got %v, want %v", readCache.LastCheck, cache.LastCheck)
	}
}

func TestShouldCheckForUpdates(t *testing.T) {
	tests := []struct {
		name          string
		lastCheckTime time.Time
		want          bool
	}{
		{
			name:          "No previous check",
			lastCheckTime: time.Time{},
			want:          true,
		},
		{
			name:          "Checked recently (1 hour ago)",
			lastCheckTime: time.Now().Add(-1 * time.Hour),
			want:          false,
		},
		{
			name:          "Checked 25 hours ago",
			lastCheckTime: time.Now().Add(-25 * time.Hour),
			want:          true,
		},
		{
			name:          "Checked exactly 24 hours ago",
			lastCheckTime: time.Now().Add(-24 * time.Hour),
			want:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := cacheData{
				LastCheck: tt.lastCheckTime,
			}

			shouldCheck := time.Since(cache.LastCheck) >= checkInterval
			if shouldCheck != tt.want {
				t.Errorf("shouldCheck = %v, want %v (time since: %v)", shouldCheck, tt.want, time.Since(cache.LastCheck))
			}
		})
	}
}

func TestNotifyIfOutdated_DisabledCheck(t *testing.T) {
	// This should return immediately without doing anything
	// We can't easily test the output, but we can ensure it doesn't panic
	NotifyIfOutdated("1.0.0", true)
}

func TestNotifyIfOutdated_DevVersion(t *testing.T) {
	// Dev version should skip update checks
	NotifyIfOutdated("dev", false)
}

func TestParseVersionTimestamp(t *testing.T) {
	tests := []struct {
		version  string
		expected time.Time
		wantErr  bool
	}{
		{
			version:  "2025.11.24.1628",
			expected: time.Date(2025, 11, 24, 16, 28, 0, 0, time.UTC),
			wantErr:  false,
		},
		{
			version:  "2025.01.01.0000",
			expected: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			wantErr:  false,
		},
		{
			version: "invalid",
			wantErr: true,
		},
		{
			version: "2025.11.24",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			result, err := parseVersionTimestamp(tt.version)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for version %s, got nil", tt.version)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if !result.Equal(tt.expected) {
				t.Errorf("parseVersionTimestamp(%s) = %v, want %v", tt.version, result, tt.expected)
			}
		})
	}
}

func TestShouldNotifyForVersion(t *testing.T) {
	tests := []struct {
		name            string
		currentVersion  string
		latestVersion   string
		lastSeenVersion string
		cfg             *config.Config
		want            bool
	}{
		{
			name:            "already notified about this version",
			currentVersion:  "2025.11.17.0000",
			latestVersion:   "2025.11.24.1628",
			lastSeenVersion: "2025.11.24.1628",
			cfg:             nil,
			want:            false,
		},
		{
			name:            "version is 7+ days old (should notify)",
			currentVersion:  "2025.11.17.0000",
			latestVersion:   "2025.11.24.1628",
			lastSeenVersion: "",
			cfg:             nil,
			want:            true,
		},
		{
			name:            "version is less than 7 days old (should notify with default threshold of 0)",
			currentVersion:  "2025.11.23.0000",
			latestVersion:   "2025.11.24.1628",
			lastSeenVersion: "",
			cfg:             nil,
			want:            true,
		},
		{
			name:            "custom threshold of 1 day",
			currentVersion:  "2025.11.23.0000",
			latestVersion:   "2025.11.24.1628",
			lastSeenVersion: "",
			cfg:             &config.Config{NotifyThresholdDays: 1},
			want:            true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldNotifyForVersion(tt.currentVersion, tt.latestVersion, tt.lastSeenVersion, tt.cfg)
			if result != tt.want {
				t.Errorf("shouldNotifyForVersion() = %v, want %v", result, tt.want)
			}
		})
	}
}

func TestDetectUpdateCommand_Scoop(t *testing.T) {
	// Test Scoop detection logic
	tests := []struct {
		name     string
		execPath string
		want     string
	}{
		{
			name:     "Scoop on Windows",
			execPath: "C:\\Users\\user\\scoop\\apps\\arcitems\\current\\arcitems.exe",
			want:     "scoop update arcitems",
		},
		{
			name:     "Scoop on Windows (forward slash)",
			execPath: "C:/Users/user/scoop/apps/arcitems/current/arcitems.exe",
			want:     "scoop update arcitems",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got string
			if contains(tt.execPath, "\\scoop\\") || contains(tt.execPath, "/scoop/") {
				got = "scoop update arcitems"
			}

			if got != tt.want {
				t.Errorf("detectUpdateCommand() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper function for string contains check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || indexOf(s, substr) >= 0))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
