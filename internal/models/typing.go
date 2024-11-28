package models

import (
	"fmt"
	"slices"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jfosburgh/keys/internal/words"
)

var (
	Untyped         = lipgloss.Color("#3e3f3e")
	Correct         = lipgloss.Color("#0f890b")
	Incorrect       = lipgloss.Color("#b21114")
	IndexBackground = lipgloss.Color("#dddddd")

	TickTimeMS = time.Millisecond * 10
	SPACE      = " "
)

type TypingTest struct {
	Target     []string
	Keystrokes []Keystroke

	StartTime time.Time

	Started   bool
	Completed bool

	WindowHeight int
	WindowWidth  int

	UnlockedLetters int
	Words           words.WordList
	K               int
	MinLength       int
}

type Keystroke struct {
	Value   string
	TypedAt time.Time
}

type Result struct {
	Characters int
	Mistakes   int
}

type TickMsg struct{}

func doTick() tea.Msg {
	time.Sleep(TickTimeMS)
	return TickMsg{}
}

func NewTypingTest() TypingTest {
	filteredWords := words.Words20k.TopK(10000).LongerThan(3).FilterOutLetter(strings.Split(words.LETTERS, "")[6:26])
	return TypingTest{
		UnlockedLetters: 6,
		Words:           filteredWords,
		K:               10000,
		MinLength:       3,

		Target: strings.Split(strings.ReplaceAll(filteredWords.TakeChars(100), " ", SPACE), ""),
	}
}

func (t *TypingTest) UpdateWordList() {
	t.Words = words.Words20k.TopK(t.K).LongerThan(t.MinLength).FilterOutLetter(strings.Split(words.LETTERS, "")[t.UnlockedLetters:26])
}

func (t *TypingTest) NewTarget() {
	t.Target = strings.Split(strings.ReplaceAll(t.Words.TakeChars(100), " ", SPACE), "")
	t.Keystrokes = []Keystroke{}
	t.Started = false
	t.Completed = false
}

func (t TypingTest) FilteredLength() int {
	letters := 0

	nonLetters := []string{"backspace", "ctrl+h", "ctrl+backspace"}

	for _, keystroke := range t.Keystrokes {
		if !slices.Contains(nonLetters, keystroke.Value) {
			letters += 1
		}
	}

	return letters
}

func (t TypingTest) Evaluate() Result {
	length := len(t.Target)
	result := Result{
		Characters: length,
		Mistakes:   t.FilteredLength() - length,
	}

	return result
}

func (t TypingTest) Typed() []string {
	characters := []string{}

	for _, keystroke := range t.Keystrokes {
		if keystroke.Value == "backspace" && len(characters) > 0 {
			characters = characters[:len(characters)-1]
			continue
		}

		if keystroke.Value == "ctrl+backspace" || keystroke.Value == "ctrl+h" {
			if len(characters) <= 1 {
				characters = []string{}
				continue
			}

			if characters[len(characters)-1] == " " {
				characters = characters[:len(characters)-1]
			}

			for len(characters) > 0 && characters[len(characters)-1] != " " {
				characters = characters[:len(characters)-1]
			}

			continue
		}

		characters = append(characters, keystroke.Value)
	}

	return strings.Split(strings.ReplaceAll(strings.Join(characters, ""), " ", SPACE), "")
}

func compare(target, test []string) bool {
	if len(target) != len(test) {
		return false
	}

	for i := range len(target) {
		if target[i] != test[i] {
			return false
		}
	}

	return true
}

func (t TypingTest) Init() tea.Cmd {
	return nil
}

func (t TypingTest) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var command tea.Cmd

	switch msg := msg.(type) {
	case TickMsg:
		return t, doTick
	case tea.WindowSizeMsg:
		t.WindowWidth = msg.Width
		t.WindowHeight = msg.Height
	case tea.KeyMsg:
		if !t.Started {
			t.Started = true
			t.StartTime = time.Now()
			command = doTick
		}

		switch msg.String() {
		case "ctrl+c", "esc":
			return t, tea.Quit
		case "ctrl+r":
			t.NewTarget()
			return t, nil
		case "+":
			t.UnlockedLetters++
			t.UpdateWordList()
			t.NewTarget()
			return t, nil
		case "-":
			t.UnlockedLetters--
			t.UpdateWordList()
			t.NewTarget()
			return t, nil
		default:
			if t.Completed {
				t.NewTarget()
				return t, nil
			}

			t.Keystrokes = append(t.Keystrokes, Keystroke{
				Value:   msg.String(),
				TypedAt: time.Now(),
			})

			t.Completed = compare(t.Target, t.Typed())
		}
	}

	return t, command
}

func (t TypingTest) View() string {
	res := ""

	typed := t.Typed()

	numTyped := len(typed)
	numTarget := len(t.Target)

	for i := range max(numTyped, len(t.Target)) {
		style := lipgloss.NewStyle().Foreground(Untyped)

		if i == numTyped {
			style = style.Background(IndexBackground)
			res += style.Render(string(t.Target[i]))
		} else if i >= numTarget {
			style = lipgloss.NewStyle().Background(Incorrect)
			res += style.Render(string(typed[i]))
		} else if i < numTyped {
			if typed[i] == t.Target[i] {
				style = lipgloss.NewStyle().Foreground(Correct)
				res += style.Render(string(typed[i]))
			} else {
				style = lipgloss.NewStyle().Background(Incorrect)
				res += style.Render(string(typed[i]))
			}
		} else {
			res += style.Render(string(t.Target[i]))
		}
	}

	letters := ""
	for i, letter := range strings.Split(words.LETTERS, "") {
		if i > 0 {
			letters += " "
		}

		if i < t.UnlockedLetters {
			letters += lipgloss.NewStyle().Foreground(Correct).Render(letter)
		} else {
			letters += lipgloss.NewStyle().Foreground(Untyped).Render(letter)
		}
	}

	res = lipgloss.NewStyle().Width(t.WindowWidth / 4 * 3).Align(lipgloss.Center).Render(res)
	res = lipgloss.Place(t.WindowWidth, t.WindowHeight-5, lipgloss.Center, lipgloss.Center, res)

	duration := time.Duration(0)
	if t.Completed {
		duration = t.Keystrokes[len(t.Keystrokes)-1].TypedAt.Sub(t.StartTime)
	} else if t.Started {
		duration = time.Since(t.StartTime)
	}

	d := duration.Round(time.Millisecond)
	minutes := d / time.Minute
	d -= minutes * time.Minute
	seconds := d / time.Second
	d -= seconds * time.Second
	milliseconds := d / time.Millisecond

	timeText := fmt.Sprintf("%02d:%02d.%03d", minutes, seconds, milliseconds)

	res = lipgloss.JoinVertical(lipgloss.Center, letters, res, timeText)

	if t.Completed {
		cpm := float64(len(t.Target)) / duration.Minutes()
		wpm := cpm / 5.

		rateStats := fmt.Sprintf("%d WPM", int(wpm))
		results := t.Evaluate()

		charStats := lipgloss.JoinHorizontal(
			lipgloss.Center,
			fmt.Sprintf("%d", results.Characters), "/",
			lipgloss.NewStyle().Foreground(Correct).Render(fmt.Sprintf("%d", results.Characters-results.Mistakes)), "/",
			lipgloss.NewStyle().Foreground(Incorrect).Render(fmt.Sprintf("%d", results.Mistakes)),
		)

		accuracy := 1.0 - float32(results.Mistakes)/float32(results.Characters)
		accuracyStats := fmt.Sprintf("%.02f%% Accuracy", accuracy*100.)

		res = lipgloss.JoinVertical(lipgloss.Center, res, rateStats, charStats, accuracyStats)
	}

	return res
}
