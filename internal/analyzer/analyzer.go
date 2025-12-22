package analyzer

import (
	"github.com/pdavlin/arcitems/internal/data"
	"github.com/pdavlin/arcitems/internal/state"
)

// ItemUsage contains analysis results for an item
type ItemUsage struct {
	Item           *data.Item
	UsedInQuests   []string         // Quest IDs that require this item
	UsedInProjects []string         // Project IDs that require this item
	UsedInHideouts map[string][]int // HideoutID -> []level numbers
	SafeToSell     bool
}

// Analyzer analyzes quest, project, and hideout requirements
type Analyzer struct {
	items    map[string]*data.Item
	quests   []*data.Quest
	projects []*data.Project
	hideouts []*data.Hideout
	// Cache for quest usage
	questUsageMap map[string][]string // itemID -> []questID
	// Cache for project usage
	projectUsageMap map[string][]string // itemID -> []projectID
	// Cache for hideout usage
	hideoutUsageMap map[string]map[string][]int // itemID -> hideoutID -> []levels
}

// NewAnalyzer creates a new analyzer with the given data
func NewAnalyzer(items map[string]*data.Item, quests []*data.Quest, projects []*data.Project, hideouts []*data.Hideout) *Analyzer {
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

// buildUsageIndex pre-computes which items are used in quests and projects
func (a *Analyzer) buildUsageIndex() {
	// Build quest usage map
	for _, quest := range a.quests {
		for _, req := range quest.RequiredItemIds {
			a.questUsageMap[req.ItemID] = append(a.questUsageMap[req.ItemID], quest.ID)
		}
	}

	// Build project usage map
	if a.projects != nil {
		for _, project := range a.projects {
			for _, phase := range project.Phases {
				// Direct item requirements
				for _, req := range phase.RequirementItemIds {
					a.projectUsageMap[req.ItemID] = append(a.projectUsageMap[req.ItemID], project.ID)
				}

				// Category-based requirements
				for _, catReq := range phase.RequirementCategories {
					// Mark all items in this category as potentially needed
					for itemID, item := range a.items {
						if item.Type == catReq.Category {
							a.projectUsageMap[itemID] = append(a.projectUsageMap[itemID], project.ID)
						}
					}
				}
			}
		}
	}

	// Build hideout usage map
	if a.hideouts != nil {
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
}

// AnalyzeItem returns usage information for a specific item
func (a *Analyzer) AnalyzeItem(itemID string) *ItemUsage {
	item, exists := a.items[itemID]
	if !exists {
		return nil
	}

	questIDs := a.questUsageMap[itemID]
	projectIDs := a.projectUsageMap[itemID]
	hideoutMap := a.hideoutUsageMap[itemID]

	// Item is safe to sell if it's not used in any quests, projects, or hideouts
	safeToSell := len(questIDs) == 0 && len(projectIDs) == 0 && len(hideoutMap) == 0

	return &ItemUsage{
		Item:           item,
		UsedInQuests:   questIDs,
		UsedInProjects: projectIDs,
		UsedInHideouts: hideoutMap,
		SafeToSell:     safeToSell,
	}
}

// AnalyzeItemWithState returns usage info considering completion state
func (a *Analyzer) AnalyzeItemWithState(itemID string, completionState *state.CompletionState) *ItemUsage {
	usage := a.AnalyzeItem(itemID)
	if usage == nil {
		return nil
	}

	// If no state, return original analysis
	if completionState == nil {
		return usage
	}

	// Filter out completed quests
	var activeQuests []string
	for _, questID := range usage.UsedInQuests {
		if !completionState.IsQuestCompleted(questID) {
			activeQuests = append(activeQuests, questID)
		}
	}

	// Filter out completed projects
	var activeProjects []string
	for _, projectID := range usage.UsedInProjects {
		if !completionState.IsProjectCompleted(projectID) {
			activeProjects = append(activeProjects, projectID)
		}
	}

	// Filter out completed hideout levels
	activeHideouts := make(map[string][]int)
	for hideoutID, levels := range usage.UsedInHideouts {
		completedLevel := completionState.GetHideoutLevel(hideoutID)

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

// GetQuestByID returns a quest by its ID
func (a *Analyzer) GetQuestByID(questID string) *data.Quest {
	for _, quest := range a.quests {
		if quest.ID == questID {
			return quest
		}
	}
	return nil
}

// GetProjectByID returns a project by its ID
func (a *Analyzer) GetProjectByID(projectID string) *data.Project {
	if a.projects == nil {
		return nil
	}
	for _, project := range a.projects {
		if project.ID == projectID {
			return project
		}
	}
	return nil
}

// GetHideoutByID returns a hideout station by its ID
func (a *Analyzer) GetHideoutByID(hideoutID string) *data.Hideout {
	if a.hideouts == nil {
		return nil
	}
	for _, hideout := range a.hideouts {
		if hideout.ID == hideoutID {
			return hideout
		}
	}
	return nil
}

// GetQuests returns all quests
func (a *Analyzer) GetQuests() []*data.Quest {
	return a.quests
}

// GetProjects returns all projects
func (a *Analyzer) GetProjects() []*data.Project {
	return a.projects
}

// GetHideouts returns all hideouts
func (a *Analyzer) GetHideouts() []*data.Hideout {
	return a.hideouts
}
