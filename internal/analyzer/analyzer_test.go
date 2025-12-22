package analyzer

import (
	"testing"

	"github.com/pdavlin/arcitems/internal/data"
	"github.com/pdavlin/arcitems/internal/state"
)

func TestAnalyzeItemWithState_HideoutCompletion(t *testing.T) {
	// Setup test data
	items := map[string]*data.Item{
		"rusted_gear": {
			ID:   "rusted_gear",
			Name: map[string]string{"en": "Rusted Gear"},
		},
	}

	hideouts := []*data.Hideout{
		{
			ID:       "weapon_bench",
			Name:     map[string]string{"en": "Weapon Bench"},
			MaxLevel: 5,
			Levels: []data.HideoutLevel{
				{Level: 1, RequirementItemIds: []data.ItemRequirement{}},
				{Level: 2, RequirementItemIds: []data.ItemRequirement{}},
				{
					Level: 3,
					RequirementItemIds: []data.ItemRequirement{
						{ItemID: "rusted_gear", Quantity: 5},
					},
				},
				{Level: 4, RequirementItemIds: []data.ItemRequirement{}},
				{Level: 5, RequirementItemIds: []data.ItemRequirement{}},
			},
		},
	}

	analyzer := NewAnalyzer(items, []*data.Quest{}, []*data.Project{}, hideouts)

	t.Run("Without state - item not safe", func(t *testing.T) {
		usage := analyzer.AnalyzeItem("rusted_gear")
		if usage == nil {
			t.Fatal("Expected usage to not be nil")
		}

		if usage.SafeToSell {
			t.Error("Expected item to NOT be safe to sell without completion state")
		}

		if len(usage.UsedInHideouts) != 1 {
			t.Errorf("Expected item to be used in 1 hideout, got %d", len(usage.UsedInHideouts))
		}

		if levels, ok := usage.UsedInHideouts["weapon_bench"]; !ok {
			t.Error("Expected item to be used in weapon_bench")
		} else if len(levels) != 1 || levels[0] != 3 {
			t.Errorf("Expected item to be used in level 3, got %v", levels)
		}
	})

	t.Run("With incomplete hideout (level 2) - item not safe", func(t *testing.T) {
		completionState := &state.CompletionState{
			HideoutLevels: map[string]int{
				"weapon_bench": 2,
			},
		}

		usage := analyzer.AnalyzeItemWithState("rusted_gear", completionState)
		if usage == nil {
			t.Fatal("Expected usage to not be nil")
		}

		if usage.SafeToSell {
			t.Error("Expected item to NOT be safe to sell with hideout at level 2")
		}

		if len(usage.UsedInHideouts) != 1 {
			t.Errorf("Expected item to still be used in 1 hideout, got %d", len(usage.UsedInHideouts))
		}

		if levels, ok := usage.UsedInHideouts["weapon_bench"]; !ok {
			t.Error("Expected item to still be needed for weapon_bench")
		} else if len(levels) != 1 || levels[0] != 3 {
			t.Errorf("Expected item to still be needed for level 3, got %v", levels)
		}
	})

	t.Run("With completed hideout (level 3) - item IS safe", func(t *testing.T) {
		completionState := &state.CompletionState{
			HideoutLevels: map[string]int{
				"weapon_bench": 3,
			},
		}

		usage := analyzer.AnalyzeItemWithState("rusted_gear", completionState)
		if usage == nil {
			t.Fatal("Expected usage to not be nil")
		}

		if !usage.SafeToSell {
			t.Error("Expected item to BE safe to sell after completing hideout level 3")
		}

		if len(usage.UsedInHideouts) != 0 {
			t.Errorf("Expected item to not be used in any hideouts, got %d hideouts: %v",
				len(usage.UsedInHideouts), usage.UsedInHideouts)
		}
	})

	t.Run("With over-completed hideout (level 5) - item IS safe", func(t *testing.T) {
		completionState := &state.CompletionState{
			HideoutLevels: map[string]int{
				"weapon_bench": 5,
			},
		}

		usage := analyzer.AnalyzeItemWithState("rusted_gear", completionState)
		if usage == nil {
			t.Fatal("Expected usage to not be nil")
		}

		if !usage.SafeToSell {
			t.Error("Expected item to BE safe to sell after maxing hideout")
		}

		if len(usage.UsedInHideouts) != 0 {
			t.Errorf("Expected item to not be used in any hideouts, got %d", len(usage.UsedInHideouts))
		}
	})
}

func TestAnalyzeItemWithState_QuestCompletion(t *testing.T) {
	// Setup test data
	items := map[string]*data.Item{
		"test_item": {
			ID:   "test_item",
			Name: map[string]string{"en": "Test Item"},
		},
	}

	quests := []*data.Quest{
		{
			ID:   "quest_1",
			Name: map[string]string{"en": "Test Quest 1"},
			RequiredItemIds: []data.ItemRequirement{
				{ItemID: "test_item", Quantity: 3},
			},
		},
		{
			ID:   "quest_2",
			Name: map[string]string{"en": "Test Quest 2"},
			RequiredItemIds: []data.ItemRequirement{
				{ItemID: "test_item", Quantity: 2},
			},
		},
	}

	analyzer := NewAnalyzer(items, quests, []*data.Project{}, []*data.Hideout{})

	t.Run("Without state - item not safe", func(t *testing.T) {
		usage := analyzer.AnalyzeItem("test_item")
		if usage == nil {
			t.Fatal("Expected usage to not be nil")
		}

		if usage.SafeToSell {
			t.Error("Expected item to NOT be safe to sell without completion state")
		}

		if len(usage.UsedInQuests) != 2 {
			t.Errorf("Expected item to be used in 2 quests, got %d", len(usage.UsedInQuests))
		}
	})

	t.Run("With one quest completed - item not safe", func(t *testing.T) {
		completionState := &state.CompletionState{
			CompletedQuests: []string{"quest_1"},
		}

		usage := analyzer.AnalyzeItemWithState("test_item", completionState)
		if usage == nil {
			t.Fatal("Expected usage to not be nil")
		}

		if usage.SafeToSell {
			t.Error("Expected item to NOT be safe to sell with only one quest completed")
		}

		if len(usage.UsedInQuests) != 1 {
			t.Errorf("Expected item to be used in 1 remaining quest, got %d", len(usage.UsedInQuests))
		}

		if usage.UsedInQuests[0] != "quest_2" {
			t.Errorf("Expected remaining quest to be quest_2, got %s", usage.UsedInQuests[0])
		}
	})

	t.Run("With all quests completed - item IS safe", func(t *testing.T) {
		completionState := &state.CompletionState{
			CompletedQuests: []string{"quest_1", "quest_2"},
		}

		usage := analyzer.AnalyzeItemWithState("test_item", completionState)
		if usage == nil {
			t.Fatal("Expected usage to not be nil")
		}

		if !usage.SafeToSell {
			t.Error("Expected item to BE safe to sell after completing all quests")
		}

		if len(usage.UsedInQuests) != 0 {
			t.Errorf("Expected item to not be used in any quests, got %d", len(usage.UsedInQuests))
		}
	})
}

func TestAnalyzeItemWithState_MultipleUsages(t *testing.T) {
	// Setup test data with an item used in both quests and hideouts
	items := map[string]*data.Item{
		"multi_use_item": {
			ID:   "multi_use_item",
			Name: map[string]string{"en": "Multi-Use Item"},
		},
	}

	quests := []*data.Quest{
		{
			ID:   "quest_a",
			Name: map[string]string{"en": "Quest A"},
			RequiredItemIds: []data.ItemRequirement{
				{ItemID: "multi_use_item", Quantity: 1},
			},
		},
	}

	hideouts := []*data.Hideout{
		{
			ID:       "station_a",
			Name:     map[string]string{"en": "Station A"},
			MaxLevel: 3,
			Levels: []data.HideoutLevel{
				{Level: 1, RequirementItemIds: []data.ItemRequirement{}},
				{
					Level: 2,
					RequirementItemIds: []data.ItemRequirement{
						{ItemID: "multi_use_item", Quantity: 2},
					},
				},
				{Level: 3, RequirementItemIds: []data.ItemRequirement{}},
			},
		},
	}

	analyzer := NewAnalyzer(items, quests, []*data.Project{}, hideouts)

	t.Run("Only quest completed - item not safe (hideout incomplete)", func(t *testing.T) {
		completionState := &state.CompletionState{
			CompletedQuests: []string{"quest_a"},
			HideoutLevels: map[string]int{
				"station_a": 1,
			},
		}

		usage := analyzer.AnalyzeItemWithState("multi_use_item", completionState)
		if usage == nil {
			t.Fatal("Expected usage to not be nil")
		}

		if usage.SafeToSell {
			t.Error("Expected item to NOT be safe - hideout still needs it")
		}

		if len(usage.UsedInQuests) != 0 {
			t.Error("Expected no active quests")
		}

		if len(usage.UsedInHideouts) != 1 {
			t.Errorf("Expected 1 active hideout, got %d", len(usage.UsedInHideouts))
		}
	})

	t.Run("Only hideout completed - item not safe (quest incomplete)", func(t *testing.T) {
		completionState := &state.CompletionState{
			CompletedQuests: []string{},
			HideoutLevels: map[string]int{
				"station_a": 2,
			},
		}

		usage := analyzer.AnalyzeItemWithState("multi_use_item", completionState)
		if usage == nil {
			t.Fatal("Expected usage to not be nil")
		}

		if usage.SafeToSell {
			t.Error("Expected item to NOT be safe - quest still needs it")
		}

		if len(usage.UsedInQuests) != 1 {
			t.Errorf("Expected 1 active quest, got %d", len(usage.UsedInQuests))
		}

		if len(usage.UsedInHideouts) != 0 {
			t.Error("Expected no active hideouts")
		}
	})

	t.Run("Both completed - item IS safe", func(t *testing.T) {
		completionState := &state.CompletionState{
			CompletedQuests: []string{"quest_a"},
			HideoutLevels: map[string]int{
				"station_a": 2,
			},
		}

		usage := analyzer.AnalyzeItemWithState("multi_use_item", completionState)
		if usage == nil {
			t.Fatal("Expected usage to not be nil")
		}

		if !usage.SafeToSell {
			t.Error("Expected item to BE safe after completing all usages")
		}

		if len(usage.UsedInQuests) != 0 {
			t.Error("Expected no active quests")
		}

		if len(usage.UsedInHideouts) != 0 {
			t.Error("Expected no active hideouts")
		}
	})
}
