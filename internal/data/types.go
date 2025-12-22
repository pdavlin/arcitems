package data

// Item represents an in-game item from ARC Raiders
type Item struct {
	ID           string            `json:"id"`
	Name         map[string]string `json:"name"`
	Description  map[string]string `json:"description,omitempty"`
	Rarity       string            `json:"rarity"`
	Type         string            `json:"type"`
	Value        float64           `json:"value"`
	RecyclesInto map[string]int    `json:"recyclesInto,omitempty"`
	SalvagesInto map[string]int    `json:"salvagesInto,omitempty"`
	FoundIn      *string           `json:"foundIn"`
	WeightKg     float64           `json:"weightKg"`
	StackSize    int               `json:"stackSize"`
	UpdatedAt    string            `json:"updatedAt,omitempty"`
}

// Quest represents a quest in ARC Raiders
type Quest struct {
	ID              string              `json:"id"`
	Name            map[string]string   `json:"name"`
	Description     map[string]string   `json:"description,omitempty"`
	Trader          string              `json:"trader,omitempty"`
	Objectives      []map[string]string `json:"objectives,omitempty"`
	RequiredItemIds []ItemRequirement   `json:"requiredItemIds,omitempty"`
	RewardItemIds   []ItemRequirement   `json:"rewardItemIds,omitempty"`
	XP              int                 `json:"xp,omitempty"`
	PreviousQuestIds []string           `json:"previousQuestIds,omitempty"`
	NextQuestIds    []string            `json:"nextQuestIds,omitempty"`
	UpdatedAt       string              `json:"updatedAt,omitempty"`
}

// ItemRequirement specifies an item and quantity needed/rewarded
type ItemRequirement struct {
	ItemID   string `json:"itemId"`
	Quantity int    `json:"quantity"`
}

// Project represents a multi-phase project
type Project struct {
	ID          string            `json:"id"`
	Name        map[string]string `json:"name,omitempty"`
	Description map[string]string `json:"description,omitempty"`
	Phases      []Phase           `json:"phases"`
}

// Phase represents a single phase within a project
type Phase struct {
	PhaseNumber           int                 `json:"phaseNumber"`
	RequirementItemIds    []ItemRequirement   `json:"requirementItemIds,omitempty"`
	RequirementCategories []CategoryReq       `json:"requirementCategories,omitempty"`
}

// CategoryReq specifies a category-based requirement
type CategoryReq struct {
	Category      string `json:"category"`
	ValueRequired int    `json:"valueRequired"`
}

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

// Metadata contains information about the data version
type Metadata struct {
	Version     string `json:"version"`
	SyncedAt    string `json:"syncedAt"`
	ItemCount   int    `json:"itemCount"`
	QuestCount  int    `json:"questCount"`
	HideoutCount int   `json:"hideoutCount"`
}
