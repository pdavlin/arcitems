package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/pdavlin/arcitems/internal/analyzer"
	"github.com/pdavlin/arcitems/internal/data"
	"github.com/pdavlin/arcitems/internal/search"
	"github.com/pdavlin/arcitems/internal/state"
	"github.com/pdavlin/arcitems/internal/ui"
	"github.com/pdavlin/arcitems/internal/update"
	"github.com/spf13/cobra"
)

var (
	// Version is set at build time
	Version = "dev"

	// Flags
	jsonOutput      bool
	interactive     bool
	lang            string
	manageMode      bool
	noStateFlag     bool
	noUpdateCheck   bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "arcitems [query]",
		Short: "ARC Raiders item and quest lookup tool",
		Long: `arcitems is a CLI tool for searching ARC Raiders items and determining
whether they are safe to sell or recycle based on quest requirements.`,
		Version: Version,
		Args:    cobra.ArbitraryArgs,
		Run:     runCommand,
	}

	rootCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output results as JSON")
	rootCmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Start interactive UI mode")
	rootCmd.Flags().StringVar(&lang, "lang", "en", "Language for item names (en, de, fr, es, etc.)")
	rootCmd.Flags().BoolVarP(&manageMode, "manage", "m", false, "Launch completion manager")
	rootCmd.Flags().BoolVar(&noStateFlag, "no-state", false, "Ignore completion state (search all items)")
	rootCmd.Flags().BoolVar(&noUpdateCheck, "no-update-check", false, "Disable update check")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runCommand(cmd *cobra.Command, args []string) {
	// Check for updates (non-blocking)
	update.NotifyIfOutdated(Version, noUpdateCheck)

	// Load embedded data
	items, quests, projects, hideouts, metadata, err := data.LoadEmbeddedData()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load data: %v\n", err)
		os.Exit(1)
	}

	// Load completion state (unless disabled)
	var completionState *state.CompletionState
	if !noStateFlag {
		var err error
		completionState, err = state.LoadState(metadata.Version)
		if err != nil {
			// Non-fatal: warn and continue without state
			fmt.Fprintf(os.Stderr, "Warning: could not load state: %v\n", err)
		}
	}

	// If manage mode, launch completion UI
	if manageMode {
		if err := ui.RunCompletion(completionState, quests, projects, hideouts); err != nil {
			fmt.Fprintf(os.Stderr, "UI error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Search mode requires query
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: query required (or use --manage)\n")
		os.Exit(1)
	}

	// Print data version info (subtle)
	if !jsonOutput {
		completedQuests := 0
		if completionState != nil {
			completedQuests = len(completionState.CompletedQuests)
		}
		fmt.Fprintf(os.Stderr, "Data version: %s (%d items, %d quests",
			metadata.Version, metadata.ItemCount, metadata.QuestCount)
		if completedQuests > 0 {
			fmt.Fprintf(os.Stderr, ", %d completed", completedQuests)
		}
		fmt.Fprintf(os.Stderr, ")\n\n")
	}

	// Create analyzer
	itemAnalyzer := analyzer.NewAnalyzer(items, quests, projects, hideouts)

	// Create searcher
	searcher := search.NewSearcher(items, itemAnalyzer, lang)

	// Perform search
	query := strings.Join(args, " ")
	results := searcher.Search(query)

	// Apply state filtering to results
	if completionState != nil {
		results = filterResultsByState(results, itemAnalyzer, completionState)
	}

	// Output results
	if jsonOutput {
		outputJSON(results)
	} else if interactive {
		// Run interactive UI
		if err := ui.Run(query, results); err != nil {
			fmt.Fprintf(os.Stderr, "UI error: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Simple list output
		outputList(results)
	}
}

func filterResultsByState(
	results []*search.SearchResult,
	itemAnalyzer *analyzer.Analyzer,
	completionState *state.CompletionState,
) []*search.SearchResult {
	filtered := make([]*search.SearchResult, len(results))
	for i, result := range results {
		// Re-analyze with state
		usage := itemAnalyzer.AnalyzeItemWithState(result.Usage.Item.ID, completionState)
		filtered[i] = &search.SearchResult{
			Usage:    usage,
			Score:    result.Score,
			MatchStr: result.MatchStr,
		}
	}
	return filtered
}

func outputJSON(results []*search.SearchResult) {
	type JSONOutput struct {
		Item           *data.Item       `json:"item"`
		UsedInQuests   []string         `json:"usedInQuests"`
		UsedInProjects []string         `json:"usedInProjects"`
		UsedInHideouts map[string][]int `json:"usedInHideouts"`
		SafeToSell     bool             `json:"safeToSell"`
	}

	output := make([]JSONOutput, len(results))
	for i, result := range results {
		output[i] = JSONOutput{
			Item:           result.Usage.Item,
			UsedInQuests:   result.Usage.UsedInQuests,
			UsedInProjects: result.Usage.UsedInProjects,
			UsedInHideouts: result.Usage.UsedInHideouts,
			SafeToSell:     result.Usage.SafeToSell,
		}
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to encode JSON: %v\n", err)
		os.Exit(1)
	}
}

func outputList(results []*search.SearchResult) {
	if len(results) == 0 {
		fmt.Println("No items found.")
		return
	}

	for _, result := range results {
		item := result.Usage.Item
		safeIcon := "✗"
		if result.Usage.SafeToSell {
			safeIcon = "✓"
		}

		fmt.Printf("%s %s [%s] %.0f coins\n",
			safeIcon,
			result.MatchStr,
			item.Rarity,
			item.Value)
	}
}
