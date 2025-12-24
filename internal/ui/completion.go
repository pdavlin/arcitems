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
	itemTypeProjectHeader // Project with progress bar
	itemTypeProjectPhase
	itemTypeHideout
)

// viewMode tracks which list is currently active
type viewMode int

const (
	viewModeQuests viewMode = iota
	viewModeProjectsHideouts
)

type completionItem struct {
	itemType    itemType
	title       string
	quest       *data.Quest
	project     *data.Project
	phaseNumber int // For project phases
	hideout     *data.Hideout
}

func (i completionItem) FilterValue() string {
	if i.itemType == itemTypeHeader || i.itemType == itemTypeProjectHeader {
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
	case itemTypeProjectHeader:
		// Calculate completed phases for this project
		completedPhases := 0
		totalPhases := len(i.project.Phases)
		for _, phase := range i.project.Phases {
			if d.completionState.IsProjectPhaseCompleted(i.project.ID, phase.PhaseNumber) {
				completedPhases++
			}
		}

		prog := progress.New(progress.WithDefaultGradient())
		prog.Width = 20
		percent := float64(completedPhases) / float64(totalPhases)
		if percent > 1.0 {
			percent = 1.0
		}
		progressBar := prog.ViewAs(percent)

		statusText := fmt.Sprintf("%d of %d phases", completedPhases, totalPhases)
		if completedPhases >= totalPhases {
			statusText += " " + safeStyle.Render("(DONE)")
		}

		projectName := i.project.Name["en"]
		if len(projectName) > 20 {
			projectName = projectName[:17] + "..."
		}
		projectName = fmt.Sprintf("%-20s", projectName)

		str = fmt.Sprintf("  %s %s %s", projectName, progressBar, statusText)
	case itemTypeQuest:
		questName := i.quest.Name["en"]
		if len(questName) > 50 {
			questName = questName[:47] + "..."
		}

		if d.completionState.IsQuestCompleted(i.quest.ID) {
			str = fmt.Sprintf("%s%s", cursor, safeStyle.Render(fmt.Sprintf("✓ %s (%s)", questName, i.quest.ID)))
		} else {
			str = fmt.Sprintf("%s• %s (%s)", cursor, questName, headerStyle.Render(i.quest.ID))
		}
	case itemTypeProjectPhase:
		// Find phase name
		var phaseName string
		for _, phase := range i.project.Phases {
			if phase.PhaseNumber == i.phaseNumber {
				if name, ok := phase.Name["en"]; ok && name != "" {
					phaseName = fmt.Sprintf("Phase %d: %s", i.phaseNumber, name)
				} else {
					phaseName = fmt.Sprintf("Phase %d", i.phaseNumber)
				}
				break
			}
		}
		if phaseName == "" {
			phaseName = fmt.Sprintf("Phase %d", i.phaseNumber)
		}

		projectName := i.project.Name["en"]
		if len(projectName) > 20 {
			projectName = projectName[:17] + "..."
		}
		displayName := fmt.Sprintf("%s - %s", projectName, phaseName)

		if d.completionState.IsProjectPhaseCompleted(i.project.ID, i.phaseNumber) {
			str = fmt.Sprintf("%s%s", cursor, safeStyle.Render(fmt.Sprintf("✓ %s", displayName)))
		} else {
			str = fmt.Sprintf("%s• %s", cursor, displayName)
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
	questList       list.Model
	projectList     list.Model
	viewMode        viewMode
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
	delegate := completionDelegate{completionState: completionState}

	// Build quest list
	questItems := buildQuestList(quests)
	questList := list.New(questItems, delegate, 0, 0)
	questList.SetShowTitle(false)
	questList.SetShowStatusBar(false)
	questList.SetFilteringEnabled(false)
	questList.SetShowHelp(false)
	selectFirstNonHeader(&questList, questItems)

	// Build projects/hideouts list
	projectItems := buildProjectHideoutList(projects, hideouts)
	projectList := list.New(projectItems, delegate, 0, 0)
	projectList.SetShowTitle(false)
	projectList.SetShowStatusBar(false)
	projectList.SetFilteringEnabled(false)
	projectList.SetShowHelp(false)
	selectFirstNonHeader(&projectList, projectItems)

	return CompletionModel{
		completionState: completionState,
		quests:          quests,
		projects:        projects,
		hideouts:        hideouts,
		questList:       questList,
		projectList:     projectList,
		viewMode:        viewModeQuests,
		saved:           false,
		ready:           false,
	}
}

func selectFirstNonHeader(l *list.Model, items []list.Item) {
	for i, item := range items {
		if ci, ok := item.(completionItem); ok && ci.itemType != itemTypeHeader && ci.itemType != itemTypeProjectHeader {
			l.Select(i)
			break
		}
	}
}

func buildQuestList(quests []*data.Quest) []list.Item {
	// Group quests by trader
	questsByTrader := make(map[string][]*data.Quest)
	for _, quest := range quests {
		trader := quest.Trader
		if trader == "" {
			trader = "Unknown"
		}
		questsByTrader[trader] = append(questsByTrader[trader], quest)
	}

	// Get sorted trader names
	var traders []string
	for trader := range questsByTrader {
		traders = append(traders, trader)
	}
	sort.Strings(traders)

	items := []list.Item{}
	for _, trader := range traders {
		traderQuests := sortQuestsByDependency(questsByTrader[trader])
		items = append(items, completionItem{itemType: itemTypeHeader, title: trader})

		for _, quest := range traderQuests {
			items = append(items, completionItem{
				itemType: itemTypeQuest,
				title:    quest.Name["en"],
				quest:    quest,
			})
		}
	}
	return items
}

func sortQuestsByDependency(traderQuests []*data.Quest) []*data.Quest {
	// Build a set of quest IDs for this trader
	questIDs := make(map[string]bool)
	for _, q := range traderQuests {
		questIDs[q.ID] = true
	}

	// Build adjacency: which quests depend on which
	dependsOn := make(map[string][]string)
	for _, q := range traderQuests {
		for _, prereq := range q.PreviousQuestIds {
			if questIDs[prereq] {
				dependsOn[q.ID] = append(dependsOn[q.ID], prereq)
			}
		}
	}

	// Kahn's algorithm for topological sort
	inDegree := make(map[string]int)
	for _, q := range traderQuests {
		inDegree[q.ID] = len(dependsOn[q.ID])
	}

	var queue []string
	for _, q := range traderQuests {
		if inDegree[q.ID] == 0 {
			queue = append(queue, q.ID)
		}
	}
	sort.Strings(queue)

	var sorted []*data.Quest
	questMap := make(map[string]*data.Quest)
	for _, q := range traderQuests {
		questMap[q.ID] = q
	}

	for len(queue) > 0 {
		sort.Strings(queue)
		id := queue[0]
		queue = queue[1:]
		sorted = append(sorted, questMap[id])

		for _, q := range traderQuests {
			for _, prereq := range dependsOn[q.ID] {
				if prereq == id {
					inDegree[q.ID]--
					if inDegree[q.ID] == 0 {
						queue = append(queue, q.ID)
					}
				}
			}
		}
	}

	if len(sorted) != len(traderQuests) {
		sort.Slice(traderQuests, func(i, j int) bool {
			return traderQuests[i].ID < traderQuests[j].ID
		})
		return traderQuests
	}

	return sorted
}

func buildProjectHideoutList(projects []*data.Project, hideouts []*data.Hideout) []list.Item {
	items := []list.Item{}

	// Add projects with phases - each project gets a header with progress bar, then its phases
	if len(projects) > 0 {
		items = append(items, completionItem{itemType: itemTypeHeader, title: "Projects"})
		for _, project := range projects {
			// Add project header with progress bar
			items = append(items, completionItem{
				itemType: itemTypeProjectHeader,
				title:    project.Name["en"],
				project:  project,
			})
			for _, phase := range project.Phases {
				items = append(items, completionItem{
					itemType:    itemTypeProjectPhase,
					title:       project.Name["en"],
					project:     project,
					phaseNumber: phase.PhaseNumber,
				})
			}
		}
	}

	// Add hideouts
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

	return items
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
			return m, tea.Quit

		case "s":
			if err := m.completionState.SaveState(); err != nil {
				// In a real app, we'd show an error message
			}
			m.saved = true
			return m, tea.Quit

		case "tab":
			// Toggle between quests and projects/hideouts view
			if m.viewMode == viewModeQuests {
				m.viewMode = viewModeProjectsHideouts
			} else {
				m.viewMode = viewModeQuests
			}
			return m, cmd

		case " ", "enter":
			m.toggleCurrent()

		case "+", "right", "=", "l":
			m.incrementHideoutLevel()

		case "-", "left", "h":
			m.decrementHideoutLevel()

		case "up", "k":
			m.moveCursorUp()
			return m, cmd

		case "down", "j":
			m.moveCursorDown()
			return m, cmd

		default:
			if m.viewMode == viewModeQuests {
				m.questList, cmd = m.questList.Update(msg)
			} else {
				m.projectList, cmd = m.projectList.Update(msg)
			}
			return m, cmd
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		headerHeight := 2 // Title + blank line
		footerHeight := 3 // Blank + help line 1 + help line 2
		verticalMargin := headerHeight + footerHeight + detailsPanelHeight

		m.questList.SetSize(msg.Width, msg.Height-verticalMargin)
		m.projectList.SetSize(msg.Width, msg.Height-verticalMargin)
		m.ready = true

		m.questList, _ = m.questList.Update(msg)
		m.projectList, cmd = m.projectList.Update(msg)
		return m, cmd
	}

	return m, cmd
}

func (m *CompletionModel) currentList() *list.Model {
	if m.viewMode == viewModeQuests {
		return &m.questList
	}
	return &m.projectList
}

func (m *CompletionModel) moveCursorUp() {
	l := m.currentList()
	prevIndex := l.Index()
	l.CursorUp()
	for i := 0; i < 100; i++ {
		currentIndex := l.Index()
		if currentIndex == prevIndex {
			break
		}
		if item, ok := l.SelectedItem().(completionItem); ok && (item.itemType == itemTypeHeader || item.itemType == itemTypeProjectHeader) {
			prevIndex = currentIndex
			l.CursorUp()
		} else {
			break
		}
	}
}

func (m *CompletionModel) moveCursorDown() {
	l := m.currentList()
	prevIndex := l.Index()
	l.CursorDown()
	for i := 0; i < 100; i++ {
		currentIndex := l.Index()
		if currentIndex == prevIndex {
			break
		}
		if item, ok := l.SelectedItem().(completionItem); ok && (item.itemType == itemTypeHeader || item.itemType == itemTypeProjectHeader) {
			prevIndex = currentIndex
			l.CursorDown()
		} else {
			break
		}
	}
}

func (m *CompletionModel) toggleCurrent() {
	l := m.currentList()
	item, ok := l.SelectedItem().(completionItem)
	if !ok {
		return
	}

	switch item.itemType {
	case itemTypeQuest:
		m.completionState.ToggleQuest(item.quest.ID)
	case itemTypeProjectPhase:
		m.completionState.ToggleProjectPhase(item.project.ID, item.phaseNumber)
	}
}

func (m *CompletionModel) incrementHideoutLevel() {
	l := m.currentList()
	item, ok := l.SelectedItem().(completionItem)
	if !ok || item.itemType != itemTypeHideout {
		return
	}

	m.completionState.IncrementHideoutLevel(item.hideout.ID, item.hideout.MaxLevel)
}

func (m *CompletionModel) decrementHideoutLevel() {
	l := m.currentList()
	item, ok := l.SelectedItem().(completionItem)
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

	// Header with view indicator
	completedQuests := len(m.completionState.CompletedQuests)
	totalQuests := len(m.quests)

	maxedHideouts := 0
	for _, hideout := range m.hideouts {
		if m.completionState.GetHideoutLevel(hideout.ID) >= hideout.MaxLevel {
			maxedHideouts++
		}
	}

	viewName := "Quests"
	if m.viewMode == viewModeProjectsHideouts {
		viewName = "Projects & Hideouts"
	}

	b.WriteString(titleStyle.Render(
		fmt.Sprintf("Completion Manager - %s (%d/%d quests, %d/%d hideouts)",
			viewName, completedQuests, totalQuests, maxedHideouts, len(m.hideouts))))
	b.WriteString("\n\n")

	// Show the active list
	if m.viewMode == viewModeQuests {
		b.WriteString(m.questList.View())

		// Quest details panel
		if item, ok := m.questList.SelectedItem().(completionItem); ok && item.itemType == itemTypeQuest {
			b.WriteString("\n")
			b.WriteString(m.renderQuestDetails(item.quest))
		} else {
			b.WriteString("\n")
			for i := 0; i < detailsPanelHeight; i++ {
				b.WriteString("\n")
			}
		}
	} else {
		b.WriteString(m.projectList.View())

		// Project phase details panel
		if item, ok := m.projectList.SelectedItem().(completionItem); ok && item.itemType == itemTypeProjectPhase {
			b.WriteString("\n")
			b.WriteString(m.renderProjectPhaseDetails(item.project, item.phaseNumber))
		} else {
			b.WriteString("\n")
			for i := 0; i < detailsPanelHeight; i++ {
				b.WriteString("\n")
			}
		}
	}

	// Footer
	b.WriteString("\n")
	b.WriteString(headerStyle.Render("Tab: switch view | Space: toggle | +/-: adjust level | ↑/↓: navigate\n"))
	b.WriteString(headerStyle.Render("s: save & exit | q: quit"))

	return b.String()
}

// Fixed height for details panel to prevent UI jumping
const detailsPanelHeight = 12

// renderProjectPhaseDetails renders details for a project phase
func (m CompletionModel) renderProjectPhaseDetails(project *data.Project, phaseNumber int) string {
	var lines []string

	lines = append(lines, headerStyle.Render(strings.Repeat("─", 60)))

	// Find the phase data
	var phase *data.Phase
	for i := range project.Phases {
		if project.Phases[i].PhaseNumber == phaseNumber {
			phase = &project.Phases[i]
			break
		}
	}

	// Project name and phase name
	phaseName := fmt.Sprintf("Phase %d", phaseNumber)
	if phase != nil {
		if name, ok := phase.Name["en"]; ok && name != "" {
			phaseName = fmt.Sprintf("Phase %d: %s", phaseNumber, name)
		}
	}
	nameLine := titleStyle.Render(fmt.Sprintf("%s - %s", project.Name["en"], phaseName))
	lines = append(lines, nameLine)

	// Phase description (prefer phase desc, fall back to project desc)
	desc := ""
	if phase != nil {
		if d, ok := phase.Description["en"]; ok && d != "" {
			desc = d
		}
	}
	if desc == "" {
		if d, ok := project.Description["en"]; ok && d != "" {
			desc = d
		}
	}
	if desc != "" {
		if len(desc) > 80 {
			desc = desc[:77] + "..."
		}
		lines = append(lines, headerStyle.Render(desc))
	} else {
		lines = append(lines, "")
	}

	if phase != nil {
		// Required items
		if len(phase.RequirementItemIds) > 0 {
			var items []string
			for i, req := range phase.RequirementItemIds {
				if i >= 4 {
					items = append(items, fmt.Sprintf("+%d more", len(phase.RequirementItemIds)-4))
					break
				}
				items = append(items, fmt.Sprintf("%dx %s", req.Quantity, req.ItemID))
			}
			lines = append(lines, unsafeStyle.Render("Required: ")+strings.Join(items, ", "))
		} else {
			lines = append(lines, "")
		}

		// Category requirements
		if len(phase.RequirementCategories) > 0 {
			var cats []string
			for _, cat := range phase.RequirementCategories {
				cats = append(cats, fmt.Sprintf("%s (%dk)", cat.Category, cat.ValueRequired/1000))
			}
			lines = append(lines, headerStyle.Render("Categories: ")+strings.Join(cats, ", "))
		} else {
			lines = append(lines, "")
		}
	} else {
		lines = append(lines, "")
		lines = append(lines, "")
	}

	// Pad to fixed height
	for len(lines) < detailsPanelHeight {
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

// renderQuestDetails renders the details panel for a selected quest
func (m CompletionModel) renderQuestDetails(quest *data.Quest) string {
	var lines []string

	// Separator line
	lines = append(lines, headerStyle.Render(strings.Repeat("─", 60)))

	// Quest name and trader
	nameLine := titleStyle.Render(quest.Name["en"])
	if quest.Trader != "" {
		nameLine += headerStyle.Render(fmt.Sprintf(" (%s)", quest.Trader))
	}
	lines = append(lines, nameLine)

	// Description
	if desc, ok := quest.Description["en"]; ok && desc != "" {
		if len(desc) > 80 {
			desc = desc[:77] + "..."
		}
		lines = append(lines, headerStyle.Render(desc))
	} else {
		lines = append(lines, "")
	}

	// Objectives (compact: show up to 2)
	if len(quest.Objectives) > 0 {
		objText := quest.Objectives[0]["en"]
		if len(objText) > 60 {
			objText = objText[:57] + "..."
		}
		line := cursorStyle.Render("Objective: ") + objText
		if len(quest.Objectives) > 1 {
			line += headerStyle.Render(fmt.Sprintf(" (+%d more)", len(quest.Objectives)-1))
		}
		lines = append(lines, line)
	} else {
		lines = append(lines, "")
	}

	// Required items (compact: inline list)
	if len(quest.RequiredItemIds) > 0 {
		var items []string
		for i, req := range quest.RequiredItemIds {
			if i >= 3 {
				items = append(items, fmt.Sprintf("+%d more", len(quest.RequiredItemIds)-3))
				break
			}
			items = append(items, fmt.Sprintf("%dx %s", req.Quantity, req.ItemID))
		}
		lines = append(lines, unsafeStyle.Render("Required: ")+strings.Join(items, ", "))
	} else {
		lines = append(lines, safeStyle.Render("No items required"))
	}

	// Rewards (compact: inline list)
	if len(quest.RewardItemIds) > 0 || quest.XP > 0 {
		var rewards []string
		if quest.XP > 0 {
			rewards = append(rewards, fmt.Sprintf("%d XP", quest.XP))
		}
		for i, reward := range quest.RewardItemIds {
			if i >= 3 {
				rewards = append(rewards, fmt.Sprintf("+%d more", len(quest.RewardItemIds)-3))
				break
			}
			rewards = append(rewards, fmt.Sprintf("%dx %s", reward.Quantity, reward.ItemID))
		}
		lines = append(lines, safeStyle.Render("Rewards: ")+strings.Join(rewards, ", "))
	} else {
		lines = append(lines, "")
	}

	// Pad to fixed height
	for len(lines) < detailsPanelHeight {
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
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
