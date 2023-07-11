package tui

import (
	"github.com/charmbracelet/bubbles/progress"
)

type model struct {
	progress progress.Model
}

func NewModel() model {
	return model{
		progress: progress.NewModel(progress.WithDefaultGradient()),
	}
}
