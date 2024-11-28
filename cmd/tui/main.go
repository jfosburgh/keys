package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jfosburgh/keys/internal/models"
	"github.com/jfosburgh/keys/internal/words"
)

func main() {
	words.InitWordLists()

	p := tea.NewProgram(models.NewTypingTest(), tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
