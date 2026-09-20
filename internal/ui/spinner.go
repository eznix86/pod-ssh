package ui

import (
	"context"
	"fmt"
	"io"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type doneMessage struct{ err error }

type spinnerModel struct {
	spinner spinner.Model
	title   string
	err     error
	done    bool
	work    func() error
}

// Spin displays an animated spinner until work completes.
func Spin(ctx context.Context, title string, output io.Writer, work func() error) error {
	model := spinnerModel{
		spinner: spinner.New(spinner.WithSpinner(spinner.Dot)),
		title:   title,
		work:    work,
	}
	model.spinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#00D7AF"))
	program := tea.NewProgram(model, tea.WithContext(ctx), tea.WithOutput(output), tea.WithEnvironment(bubbleTeaEnvironment()))
	result, err := program.Run()
	if err != nil {
		return fmt.Errorf("run progress display: %w", err)
	}
	finalModel, ok := result.(spinnerModel)
	if !ok {
		return errorsNewUnexpectedModel()
	}
	return finalModel.err
}

func (m spinnerModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, func() tea.Msg { return doneMessage{err: m.work()} })
}

func (m spinnerModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := message.(type) {
	case doneMessage:
		m.err = typed.err
		m.done = true
		return m, tea.Quit
	case spinner.TickMsg:
		var command tea.Cmd
		m.spinner, command = m.spinner.Update(message)
		return m, command
	}
	return m, nil
}

func (m spinnerModel) View() tea.View {
	if m.done {
		return tea.NewView("")
	}
	return tea.NewView(fmt.Sprintf("%s %s", m.spinner.View(), m.title))
}

func errorsNewUnexpectedModel() error {
	return fmt.Errorf("progress display returned an unexpected model")
}
