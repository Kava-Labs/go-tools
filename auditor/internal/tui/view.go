package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	padding  = 2
	maxWidth = 80
)

var helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render

func (m model) View() string {
	pad := strings.Repeat(" ", padding)

	if m.progress.Percent() == 1.0 {
		return "\n" +
			pad + m.progress.View() + "\n\n" +
			pad + helpStyle("I'm done now, Bye!")
	}

	return "\n" +
		pad + m.progress.View() + "\n\n" +
		pad + helpStyle("Press q to quit")
}
