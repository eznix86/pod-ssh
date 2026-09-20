// Package ui provides terminal interactions for pod-ssh.
package ui

import (
	"errors"
	"fmt"
	"io"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// ErrCancelled is returned when the user closes a picker without selecting an item.
var ErrCancelled = errors.New("selection cancelled")

// Item is a searchable picker entry.
type Item struct {
	Name   string
	Detail string
}

func (i Item) FilterValue() string { return i.Name + " " + i.Detail }
func (i Item) Title() string       { return i.Name }
func (i Item) Description() string { return i.Detail }

type pickerModel struct {
	list      list.Model
	selected  *Item
	cancelled bool
}

// Select displays a fuzzy-searchable, single-select list.
func Select(title string, items []Item, input io.Reader, output io.Writer) (Item, error) {
	if len(items) == 0 {
		return Item{}, fmt.Errorf("%s: no choices", title)
	}
	listItems := make([]list.Item, 0, len(items))
	for _, item := range items {
		listItems = append(listItems, item)
	}

	delegate := list.NewDefaultDelegate()
	delegate.SetSpacing(0)
	delegate.Styles.SelectedTitle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(lipgloss.Color("#00D7AF")).
		Foreground(lipgloss.Color("#00D7AF")).
		Padding(0, 0, 0, 1)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedTitle.Foreground(lipgloss.Color("#7DDFCA"))
	model := pickerModel{list: list.New(listItems, delegate, 80, 18)}
	model.list.Title = title
	model.list.Styles.Title = lipgloss.NewStyle().Foreground(lipgloss.Color("#00D7AF")).Bold(true)
	model.list.Styles.Filter.Cursor.Color = lipgloss.Color("#00D7AF")
	model.list.SetShowHelp(true)
	model.list.SetFilteringEnabled(true)
	model.list.DisableQuitKeybindings()

	program := tea.NewProgram(model, tea.WithInput(input), tea.WithOutput(output), tea.WithEnvironment(bubbleTeaEnvironment()))
	result, err := program.Run()
	if err != nil {
		return Item{}, fmt.Errorf("run %s picker: %w", title, err)
	}
	finalModel, ok := result.(pickerModel)
	if !ok || finalModel.cancelled || finalModel.selected == nil {
		return Item{}, ErrCancelled
	}
	return *finalModel.selected, nil
}

func (m pickerModel) Init() tea.Cmd { return nil }

func (m pickerModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := message.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(typed.Width, typed.Height-1)
	case tea.KeyPressMsg:
		switch typed.String() {
		case "ctrl+c", "esc":
			if !m.list.SettingFilter() {
				m.cancelled = true
				return m, tea.Quit
			}
		case "enter":
			if selected, ok := m.list.SelectedItem().(Item); ok {
				m.selected = &selected
				return m, tea.Quit
			}
		}
	}

	var command tea.Cmd
	m.list, command = m.list.Update(message)
	return m, command
}

func (m pickerModel) View() tea.View {
	view := tea.NewView(m.list.View())
	view.AltScreen = true
	return view
}
