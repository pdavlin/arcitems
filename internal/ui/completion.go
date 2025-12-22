package ui

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/pdavlin/arcitems/internal/data"
	"github.com/pdavlin/arcitems/internal/state"
)

// Item wrapper types for list component

type itemType int

const (
	itemTypeHeader itemType = iota
	itemTypeQuest
	itemTypeProject
	itemTypeHideout
)

type completionItem struct {
	itemType itemType
	title    string
	quest    *data.Quest
	project  *data.Project
	hideout  *data.Hideout
}

func (i completionItem) FilterValue() string {
	if i.itemType == itemTypeHeader {
		return ""
	}
	return i.title
}

// Custom delegate for rendering completion items
type completionDelegate struct {
	completionState *state.CompletionState
}

func (d completionDelegate) Height() int                             { return 1 }
func (d completionDelegate) Spacing() int                            { return 0 }
func (d completionDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d completionDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	i, ok := item.(completionItem)
	if !ok {
		return
	}

	var str string
	cursor := "  "
	if index == m.Index() {
		cursor = cursorStyle.Render("→ ")
	}

	switch i.itemType {
	case itemTypeHeader:
		if i.title == "Quests" || i.title == "Projects" || i.title == "Hideout Stations" {
			str = titleStyle.Render(i.title)
		} else {
			str = headerStyle.Render("  " + i.title)
		}
	case itemTypeQuest:
		questName := i.quest.Name["en"]
		if len(questName) > 50 {
			questName = questName[:47] + "..."
		}

		if d.completionState.IsQuestCompleted(i.quest.ID) {
			// Entire line green when completed
			str = fmt.Sprintf("%s%s", cursor, safeStyle.Render(fmt.Sprintf("✓ %s (%s)", questName, i.quest.ID)))
		} else {
			str = fmt.Sprintf("%s• %s (%s)", cursor, questName, headerStyle.Render(i.quest.ID))
		}
	case itemTypeProject:
		projectName := i.project.Name["en"]
		if len(projectName) > 50 {
			projectName = projectName[:47] + "..."
		}

		if d.completionState.IsProjectCompleted(i.project.ID) {
			// Entire line green when completed
			str = fmt.Sprintf("%s%s", cursor, safeStyle.Render(fmt.Sprintf("✓ %s", projectName)))
		} else {
			str = fmt.Sprintf("%s• %s", cursor, projectName)
		}
	case itemTypeHideout:
		currentLevel := d.completionState.GetHideoutLevel(i.hideout.ID)
		maxLevel := i.hideout.MaxLevel

		prog := progress.New(progress.WithDefaultGradient())
		prog.Width = 20
		percent := float64(currentLevel) / float64(maxLevel)
		if percent > 1.0 {
			percent = 1.0
		}
		progressBar := prog.ViewAs(percent)

		statusText := fmt.Sprintf("Level %d of %d", currentLevel, maxLevel)
		if currentLevel >= maxLevel {
			statusText += " " + safeStyle.Render("(MAX)")
		}

		stationName := i.hideout.Name["en"]
		if len(stationName) > 20 {
			stationName = stationName[:17] + "..."
		}
		stationName = fmt.Sprintf("%-20s", stationName)

		str = fmt.Sprintf("%s%s %s %s", cursor, stationName, progressBar, statusText)
	}

	fmt.Fprint(w, str)
}

// CompletionModel represents the completion manager UI
type CompletionModel struct {
	completionState *state.CompletionState
	quests          []*data.Quest
	projects        []*data.Project
	hideouts        []*data.Hideout
	list            list.Model
	saved           bool
	width           int
	height          int
	ready           bool
}

// NewCompletionModel creates a new completion manager model
func NewCompletionModel(
	completionState *state.CompletionState,
	quests []*data.Quest,
	projects []*data.Project,
	hideouts []*data.Hideout,
) CompletionModel {
	// Sort quests by trader, then by ID for consistent display and grouping
	sort.Slice(quests, func(i, j int) bool {
		traderI := quests[i].Trader
		traderJ := quests[j].Trader
		if traderI == "" {
			traderI = "Unknown"
		}
		if traderJ == "" {
			traderJ = "Unknown"
		}
		if traderI != traderJ {
			return traderI < traderJ
		}
		return quests[i].ID < quests[j].ID
	})

	// Build flat list of items with headers
	items := []list.Item{}

	// Add quests section
	if len(quests) > 0 {
		items = append(items, completionItem{itemType: itemTypeHeader, title: "Quests"})

		lastTrader := ""
		for _, quest := range quests {
			if quest.Trader != lastTrader {
				traderName := quest.Trader
				if traderName == "" {
					traderName = "Unknown"
				}
				items = append(items, completionItem{itemType: itemTypeHeader, title: traderName})
				lastTrader = quest.Trader
			}
			items = append(items, completionItem{
				itemType: itemTypeQuest,
				title:    quest.Name["en"],
				quest:    quest,
			})
		}
	}

	// Add projects section
	if len(projects) > 0 {
		items = append(items, completionItem{itemType: itemTypeHeader, title: "Projects"})
		for _, project := range projects {
			items = append(items, completionItem{
				itemType: itemTypeProject,
				title:    project.Name["en"],
				project:  project,
			})
		}
	}

	// Add hideouts section
	if len(hideouts) > 0 {
		items = append(items, completionItem{itemType: itemTypeHeader, title: "Hideout Stations"})
		for _, hideout := range hideouts {
			if hideout.MaxLevel == 0 {
				continue
			}
			items = append(items, completionItem{
				itemType: itemTypeHideout,
				title:    hideout.Name["en"],
				hideout:  hideout,
			})
		}
	}

	// Create list with custom delegate
	delegate := completionDelegate{completionState: completionState}
	l := list.New(items, delegate, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)

	// Skip to first non-header item
	for i, item := range items {
		if ci, ok := item.(completionItem); ok && ci.itemType != itemTypeHeader {
			l.Select(i)
			break
		}
	}

	return CompletionModel{
		completionState: completionState,
		quests:          quests,
		projects:        projects,
		hideouts:        hideouts,
		list:            l,
		saved:           false,
		ready:           false,
	}
}

func (m CompletionModel) Init() tea.Cmd {
	return tea.WindowSize()
}

func (m CompletionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			if !m.saved {
				// Could warn about unsaved changes, but for now just quit
			}
			return m, tea.Quit

		case "s":
			// Save and exit
			if err := m.completionState.SaveState(); err != nil {
				// In a real app, we'd show an error message
			}
			m.saved = true
			return m, tea.Quit

		case " ", "enter":
			// Toggle current item
			m.toggleCurrent()

		case "+", "right", "=", "l":
			// Increment hideout level
			m.incrementHideoutLevel()

		case "-", "left", "h":
			// Decrement hideout level
			m.decrementHideoutLevel()

		case "up", "k":
			// Skip over headers when moving up
			prevIndex := m.list.Index()
			m.list.CursorUp()
			// Prevent infinite loop: only skip headers if we're actually moving
			for i := 0; i < 100; i++ {
				currentIndex := m.list.Index()
				if currentIndex == prevIndex {
					// Cursor didn't move, we're at a boundary
					break
				}
				if item, ok := m.list.SelectedItem().(completionItem); ok && item.itemType == itemTypeHeader {
					prevIndex = currentIndex
					m.list.CursorUp()
				} else {
					break
				}
			}
			return m, cmd

		case "down", "j":
			// Skip over headers when moving down
			prevIndex := m.list.Index()
			m.list.CursorDown()
			// Prevent infinite loop: only skip headers if we're actually moving
			for i := 0; i < 100; i++ {
				currentIndex := m.list.Index()
				if currentIndex == prevIndex {
					// Cursor didn't move, we're at a boundary
					break
				}
				if item, ok := m.list.SelectedItem().(completionItem); ok && item.itemType == itemTypeHeader {
					prevIndex = currentIndex
					m.list.CursorDown()
				} else {
					break
				}
			}
			return m, cmd

		default:
			// Let list handle other keys
			m.list, cmd = m.list.Update(msg)
			return m, cmd
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		headerHeight := 2 // Title + blank line
		footerHeight := 3 // Blank + help line 1 + help line 2
		verticalMargin := headerHeight + footerHeight

		m.list.SetSize(msg.Width, msg.Height-verticalMargin)
		m.ready = true

		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}

	return m, cmd
}

func (m *CompletionModel) toggleCurrent() {
	item, ok := m.list.SelectedItem().(completionItem)
	if !ok {
		return
	}

	switch item.itemType {
	case itemTypeQuest:
		m.completionState.ToggleQuest(item.quest.ID)
	case itemTypeProject:
		m.completionState.ToggleProject(item.project.ID)
	}
}

func (m *CompletionModel) incrementHideoutLevel() {
	item, ok := m.list.SelectedItem().(completionItem)
	if !ok || item.itemType != itemTypeHideout {
		return
	}

	m.completionState.IncrementHideoutLevel(item.hideout.ID, item.hideout.MaxLevel)
}

func (m *CompletionModel) decrementHideoutLevel() {
	item, ok := m.list.SelectedItem().(completionItem)
	if !ok || item.itemType != itemTypeHideout {
		return
	}

	m.completionState.DecrementHideoutLevel(item.hideout.ID)
}

func (m CompletionModel) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	var b strings.Builder

	// Header (always visible)
	completedQuests := len(m.completionState.CompletedQuests)
	totalQuests := len(m.quests)

	// Count maxed hideouts
	maxedHideouts := 0
	for _, hideout := range m.hideouts {
		if m.completionState.GetHideoutLevel(hideout.ID) >= hideout.MaxLevel {
			maxedHideouts++
		}
	}

	b.WriteString(titleStyle.Render(
		fmt.Sprintf("Completion Manager (%d of %d quests, %d/%d hideouts maxed)",
			completedQuests, totalQuests, maxedHideouts, len(m.hideouts))))
	b.WriteString("\n\n")

	// List (handles scrolling and rendering)
	b.WriteString(m.list.View())

	// Footer (always visible)
	b.WriteString("\n\n")
	b.WriteString(headerStyle.Render("Space: toggle | +/-: adjust level | ↑/↓: navigate\n"))
	b.WriteString(headerStyle.Render("s: save & exit | q: quit"))

	return b.String()
}

// RunCompletion runs the completion manager UI
func RunCompletion(
	completionState *state.CompletionState,
	quests []*data.Quest,
	projects []*data.Project,
	hideouts []*data.Hideout,
) error {
	p := tea.NewProgram(
		NewCompletionModel(completionState, quests, projects, hideouts),
		tea.WithAltScreen(), // Enable alternate screen buffer
	)
	_, err := p.Run()
	return err
}
