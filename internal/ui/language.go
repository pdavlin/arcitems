package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// Supported languages with native names
var supportedLanguages = []struct {
	Code string
	Name string
}{
	{"en", "English"},
	{"de", "Deutsch"},
	{"fr", "Francais"},
	{"es", "Espanol"},
	{"pt", "Portugues"},
	{"it", "Italiano"},
	{"pl", "Polski"},
	{"no", "Norsk"},
	{"da", "Dansk"},
	{"ru", "Russkiy"},
	{"uk", "Ukrayinska"},
	{"tr", "Turkce"},
	{"hr", "Hrvatski"},
	{"sr", "Srpski"},
	{"ja", "Nihongo"},
	{"kr", "Hangugeo"},
	{"zh-CN", "Simplified Chinese"},
	{"zh-TW", "Traditional Chinese"},
}

// languageItem represents a language option in the list
type languageItem struct {
	code string
	name string
}

func (i languageItem) FilterValue() string { return i.name }

// languageDelegate handles rendering of language items
type languageDelegate struct{}

func (d languageDelegate) Height() int                             { return 1 }
func (d languageDelegate) Spacing() int                            { return 0 }
func (d languageDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d languageDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	i, ok := item.(languageItem)
	if !ok {
		return
	}

	cursor := "  "
	if index == m.Index() {
		cursor = cursorStyle.Render("> ")
	}

	str := fmt.Sprintf("%s%-25s %s", cursor, i.name, headerStyle.Render("("+i.code+")"))
	fmt.Fprint(w, str)
}

// LanguagePickerModel represents the language picker UI
type LanguagePickerModel struct {
	list     list.Model
	selected string
	quit     bool
	width    int
	height   int
	ready    bool
}

// NewLanguagePickerModel creates a new language picker model
func NewLanguagePickerModel() LanguagePickerModel {
	items := make([]list.Item, len(supportedLanguages))
	for i, lang := range supportedLanguages {
		items[i] = languageItem{code: lang.Code, name: lang.Name}
	}

	delegate := languageDelegate{}
	l := list.New(items, delegate, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)

	return LanguagePickerModel{
		list:     l,
		selected: "",
		quit:     false,
		ready:    false,
	}
}

func (m LanguagePickerModel) Init() tea.Cmd {
	return tea.WindowSize()
}

func (m LanguagePickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.quit = true
			return m, tea.Quit

		case "enter", " ":
			if item, ok := m.list.SelectedItem().(languageItem); ok {
				m.selected = item.code
			}
			return m, tea.Quit

		case "up", "k":
			m.list.CursorUp()
			return m, cmd

		case "down", "j":
			m.list.CursorDown()
			return m, cmd

		default:
			m.list, cmd = m.list.Update(msg)
			return m, cmd
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		headerHeight := 3 // Title + subtitle + blank line
		footerHeight := 2 // Blank + help line
		verticalMargin := headerHeight + footerHeight

		m.list.SetSize(msg.Width, msg.Height-verticalMargin)
		m.ready = true

		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}

	return m, cmd
}

func (m LanguagePickerModel) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	var b strings.Builder

	b.WriteString(titleStyle.Render("Select your preferred language"))
	b.WriteString("\n")
	b.WriteString(headerStyle.Render("This will be used for item names in search results"))
	b.WriteString("\n\n")

	b.WriteString(m.list.View())

	b.WriteString("\n\n")
	b.WriteString(headerStyle.Render("Enter: select | q: quit (defaults to English)"))

	return b.String()
}

// Selected returns the selected language code
func (m LanguagePickerModel) Selected() string {
	return m.selected
}

// Quit returns true if the user quit without selecting
func (m LanguagePickerModel) Quit() bool {
	return m.quit
}

// RunLanguagePicker runs the language picker and returns the selected language code.
// Returns "en" if the user quits without selecting or if an error occurs.
func RunLanguagePicker() (string, error) {
	model := NewLanguagePickerModel()
	p := tea.NewProgram(model, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return "en", err
	}

	if m, ok := finalModel.(LanguagePickerModel); ok {
		if m.quit || m.selected == "" {
			return "en", nil
		}
		return m.selected, nil
	}

	return "en", nil
}
