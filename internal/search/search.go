package search

import (
	"sort"
	"strings"

	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/pdavlin/arcitems/internal/analyzer"
	"github.com/pdavlin/arcitems/internal/data"
)

// SearchResult represents a search result with ranking
type SearchResult struct {
	Usage    *analyzer.ItemUsage
	Score    int    // Lower is better (Levenshtein distance)
	MatchStr string // The matched string for display
}

// Searcher handles fuzzy searching of items
type Searcher struct {
	items    map[string]*data.Item
	analyzer *analyzer.Analyzer
	lang     string // Language code for search (e.g., "en")
}

// NewSearcher creates a new searcher
func NewSearcher(items map[string]*data.Item, analyzer *analyzer.Analyzer, lang string) *Searcher {
	if lang == "" {
		lang = "en"
	}
	return &Searcher{
		items:    items,
		analyzer: analyzer,
		lang:     lang,
	}
}

// Search performs fuzzy search on item names
func (s *Searcher) Search(query string) []*SearchResult {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return nil
	}

	var results []*SearchResult

	for itemID, item := range s.items {
		// Get item name in the specified language
		name, exists := item.Name[s.lang]
		if !exists {
			name = item.Name["en"] // Fallback to English
		}

		nameLower := strings.ToLower(name)

		// Check if query matches using fuzzy search
		if fuzzy.Match(query, nameLower) {
			// Calculate Levenshtein distance for ranking
			distance := levenshtein(query, nameLower)

			// Get usage information
			usage := s.analyzer.AnalyzeItem(itemID)

			results = append(results, &SearchResult{
				Usage:    usage,
				Score:    distance,
				MatchStr: name,
			})
		}
	}

	// Sort by distance (lower is better)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score < results[j].Score
	})

	return results
}

// levenshtein calculates the Levenshtein distance between two strings
func levenshtein(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	// Create a 2D matrix
	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
		matrix[i][0] = i
	}
	for j := range matrix[0] {
		matrix[0][j] = j
	}

	// Fill the matrix
	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 0
			if s1[i-1] != s2[j-1] {
				cost = 1
			}
			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}
