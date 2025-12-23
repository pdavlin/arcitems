package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pdavlin/arcitems/internal/search"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("86"))

	safeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("46"))

	unsafeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	cursorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			Bold(true)

	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	rarityColors = map[string]string{
		"Common":    "250",
		"Rare":      "39",
		"Epic":      "129",
		"Legendary": "214",
	}
)

// Model represents the Bubble Tea model
type Model struct {
	searchQuery string
	results     []*search.SearchResult
	cursor      int
	width       int
	height      int
}

// NewModel creates a new UI model
func NewModel(query string, results []*search.SearchResult) Model {
	return Model{
		searchQuery: query,
		results:     results,
		cursor:      0,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.results)-1 {
				m.cursor++
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

func (m Model) View() string {
	var b strings.Builder

	// Header
	b.WriteString(headerStyle.Render(fmt.Sprintf("Search: %s", m.searchQuery)))
	b.WriteString("\n\n")

	if len(m.results) == 0 {
		b.WriteString("No items found.\n")
		b.WriteString(headerStyle.Render("\nPress q to quit"))
		return b.String()
	}

	// Results header
	b.WriteString(headerStyle.Render(fmt.Sprintf("Found %d match(es)\n\n", len(m.results))))

	// Display results
	for i, result := range m.results {
		item := result.Usage.Item
		cursor := "  "
		if i == m.cursor {
			cursor = cursorStyle.Render("● ")
		} else {
			cursor = "  "
		}

		// Safe/unsafe indicator
		var safeIcon string
		if result.Usage.SafeToSell {
			safeIcon = safeStyle.Render("✓")
		} else {
			safeIcon = unsafeStyle.Render("✗")
		}

		// Item name with rarity color
		rarityColor := rarityColors[item.Rarity]
		if rarityColor == "" {
			rarityColor = "250"
		}
		nameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(rarityColor)).Bold(i == m.cursor)
		itemName := nameStyle.Render(result.MatchStr)

		b.WriteString(fmt.Sprintf("%s%s %s\n", cursor, itemName, safeIcon))

		// Show details for selected item
		if i == m.cursor {
			b.WriteString(fmt.Sprintf("  %s | %.0f coins\n",
				item.Rarity, item.Value))

			// Quest usage
			if len(result.Usage.UsedInQuests) > 0 {
				b.WriteString(unsafeStyle.Render(fmt.Sprintf("  ⚠ Required by %d quest(s)\n", len(result.Usage.UsedInQuests))))
			} else if len(result.Usage.UsedInProjects) > 0 {
				b.WriteString(unsafeStyle.Render(fmt.Sprintf("  ⚠ Required by %d project(s)\n", len(result.Usage.UsedInProjects))))
			} else {
				b.WriteString(safeStyle.Render("  ✓ Safe to sell/recycle\n"))
			}

			// Recycle info
			if len(item.RecyclesInto) > 0 {
				b.WriteString("  Recycles into: ")
				first := true
				for matID, qty := range item.RecyclesInto {
					if !first {
						b.WriteString(", ")
					}
					b.WriteString(fmt.Sprintf("%dx %s", qty, matID))
					first = false
				}
				b.WriteString("\n")
			}

			// Salvage info
			if len(item.SalvagesInto) > 0 {
				b.WriteString("  Salvages into: ")
				first := true
				for matID, qty := range item.SalvagesInto {
					if !first {
						b.WriteString(", ")
					}
					b.WriteString(fmt.Sprintf("%dx %s", qty, matID))
					first = false
				}
				b.WriteString("\n")
			}

			b.WriteString("\n")
		}
	}

	// Footer
	b.WriteString("\n")
	b.WriteString(headerStyle.Render("↑/k: up | ↓/j: down | q: quit"))

	return b.String()
}

// Run runs the Bubble Tea program
func Run(query string, results []*search.SearchResult) error {
	p := tea.NewProgram(NewModel(query, results))
	_, err := p.Run()
	return err
}
