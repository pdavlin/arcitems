package data

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed bundled/items.json
var itemsJSON []byte

//go:embed bundled/quests.json
var questsJSON []byte

//go:embed bundled/projects.json
var projectsJSON []byte

//go:embed bundled/metadata.json
var metadataJSON []byte

//go:embed bundled/hideouts.json
var hideoutsJSON []byte

// LoadEmbeddedData loads all embedded JSON data into memory
func LoadEmbeddedData() (map[string]*Item, []*Quest, []*Project, []*Hideout, *Metadata, error) {
	// Parse items
	var itemList []Item
	if err := json.Unmarshal(itemsJSON, &itemList); err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("failed to parse items: %w", err)
	}

	// Convert items list to map keyed by ID
	items := make(map[string]*Item, len(itemList))
	for i := range itemList {
		items[itemList[i].ID] = &itemList[i]
	}

	// Parse quests
	var quests []*Quest
	if err := json.Unmarshal(questsJSON, &quests); err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("failed to parse quests: %w", err)
	}

	// Parse projects
	var projects []*Project
	if err := json.Unmarshal(projectsJSON, &projects); err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("failed to parse projects: %w", err)
	}

	// Parse hideouts
	var hideouts []*Hideout
	if err := json.Unmarshal(hideoutsJSON, &hideouts); err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("failed to parse hideouts: %w", err)
	}

	// Parse metadata
	var metadata Metadata
	if err := json.Unmarshal(metadataJSON, &metadata); err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	return items, quests, projects, hideouts, &metadata, nil
}
