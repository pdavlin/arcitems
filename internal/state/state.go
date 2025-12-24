package state

import (
	"encoding/json"
	"fmt"
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

// IsProjectCompleted checks if a project is marked complete (all phases)
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

// IsProjectPhaseCompleted checks if a specific project phase is completed
func (s *CompletionState) IsProjectPhaseCompleted(projectID string, phase int) bool {
	phaseKey := fmt.Sprintf("%s:%d", projectID, phase)
	for _, id := range s.CompletedProjects {
		if id == phaseKey {
			return true
		}
	}
	return false
}

// ToggleProjectPhase toggles completion status for a project phase
func (s *CompletionState) ToggleProjectPhase(projectID string, phase int) {
	phaseKey := fmt.Sprintf("%s:%d", projectID, phase)
	for i, id := range s.CompletedProjects {
		if id == phaseKey {
			s.CompletedProjects = append(s.CompletedProjects[:i], s.CompletedProjects[i+1:]...)
			return
		}
	}
	s.CompletedProjects = append(s.CompletedProjects, phaseKey)
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
