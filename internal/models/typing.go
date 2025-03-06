package models

import (
	"fmt"
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
	Fixed           = lipgloss.Color("#d8532e")
	IndexBackground = lipgloss.Color("#dddddd")

	TickTimeMS = time.Millisecond * 10
	SPACE      = " "
)

type TypingTest struct {
	Target     []string
	Keystrokes []Keystroke
	Index      int

	StartTime time.Time

	Started   bool
	Completed bool

	WindowHeight int
	WindowWidth  int

	UnlockedLetters int
	Words           words.WordList
	K               int
	MinLength       int

	LessonLength int

	WPMTarget int
}

type KeystrokeStatus uint

const (
	UNTYPED KeystrokeStatus = iota
	CORRECT
	ERROR_UNFIXED
	ERROR_FIXED
)

type Keystroke struct {
	ExpectedValue string
	Value         string
	TypedAt       time.Time
	Status        KeystrokeStatus
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
	t := TypingTest{
		UnlockedLetters: 6,
		K:               10000,
		MinLength:       4,
		LessonLength:    125,

		WPMTarget: 35,
	}

	t.UpdateWordList()
	t.NewTarget()

	return t
}

func (t *TypingTest) UpdateWordList() {
	t.Words = words.Words20k.TopK(t.K).LongerThan(t.MinLength).FilterOutLetter(strings.Split(words.LETTERS, "")[t.UnlockedLetters:26])
}

func (t *TypingTest) NewTarget() {
	t.Index = 0
	t.Target = strings.Split(strings.ReplaceAll(t.Words.TakeChars(t.LessonLength), " ", SPACE), "")
	t.Keystrokes = make([]Keystroke, len(t.Target))
	for i := range t.Target {
		t.Keystrokes[i].ExpectedValue = t.Target[i]
	}
	t.Started = false
	t.Completed = false
}

func (t *TypingTest) ProcessKeystroke(key string) {
	if (strings.Contains(key, "backspace") || key == "ctrl+h") && t.Index <= 1 {
		t.Keystrokes[0].Value = ""
		t.Index = 0
		return
	}

	if key == "backspace" {
		t.Index -= 1
		t.Keystrokes[t.Index].Value = ""
		return
	} else if key == "ctrl+backspace" || key == "ctrl+h" {
		t.Index -= 1
		for t.Keystrokes[t.Index].Value != SPACE {
			t.Keystrokes[t.Index].Value = ""
			if t.Index == 0 {
				return
			}

			t.Index -= 1
		}
	}

	if key == " " {
		t.Keystrokes[t.Index].Value = SPACE
	} else {
		t.Keystrokes[t.Index].Value = key
	}

	t.Keystrokes[t.Index].TypedAt = time.Now()

	correct := t.Keystrokes[t.Index].Value == t.Keystrokes[t.Index].ExpectedValue
	switch t.Keystrokes[t.Index].Status {
	case UNTYPED:
		if correct {
			t.Keystrokes[t.Index].Status = CORRECT
		} else {
			t.Keystrokes[t.Index].Status = ERROR_UNFIXED
		}
	case CORRECT:
		if !correct {
			t.Keystrokes[t.Index].Status = ERROR_UNFIXED
		}
	case ERROR_UNFIXED:
		if correct {
			t.Keystrokes[t.Index].Status = ERROR_FIXED
		}
	case ERROR_FIXED:
		if !correct {
			t.Keystrokes[t.Index].Status = ERROR_UNFIXED
		}
	}

	t.Completed = t.Keystrokes[len(t.Keystrokes)-1].Status != UNTYPED

	t.Index += 1
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

			t.ProcessKeystroke(msg.String())
		}
	}

	return t, command
}

func (t TypingTest) View() string {
	res := ""

	for i := range len(t.Keystrokes) {
		style := lipgloss.NewStyle().Foreground(Untyped)
		key := t.Keystrokes[i].ExpectedValue

		if i == t.Index {
			style = style.Foreground(IndexBackground).Underline(true)
		} else if i < t.Index {
			key = t.Keystrokes[i].Value
			switch t.Keystrokes[i].Status {
			case CORRECT:
				style = style.Foreground(Correct)
			case ERROR_FIXED:
				style = style.Foreground(Fixed)
			case ERROR_UNFIXED:
				style = style.Background(Incorrect).Foreground(IndexBackground)
			}
		}
		res += style.Render(key)
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

	max_w := max(1, t.WindowWidth/4) * 3
	w, _ := lipgloss.Size(res)
	n_lines := max(3, w/max_w)

	char_per_line := w / n_lines

	words := strings.Split(res, SPACE)
	res = ""
	line := " "
	c_in_line := 0
	for _, word := range words {
		word_width, _ := lipgloss.Size(word)
		c_in_line += word_width

		if line != " " {
			line += SPACE
		}

		line += word

		if c_in_line >= char_per_line {
			line += " "
			if res == "" {
				res = line
			} else {
				res = lipgloss.JoinVertical(lipgloss.Left, res, line)
			}

			c_in_line = 0
			line = " "
		}
	}
	res = lipgloss.JoinVertical(lipgloss.Left, res, line)
	res = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).Foreground(lipgloss.Color("#000000")).Render(res)

	res = lipgloss.Place(t.WindowWidth, t.WindowHeight-6, lipgloss.Center, lipgloss.Center, res, lipgloss.WithWhitespaceBackground(lipgloss.NewStyle().GetBackground()))

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

	res = lipgloss.JoinVertical(lipgloss.Center, fmt.Sprintf("Target: %d WPM", t.WPMTarget), letters, res, timeText)

	if t.Completed {
		cpm := float64(len(t.Target)) / duration.Minutes()
		wpm := cpm / 5.

		rateStats := fmt.Sprintf("%d WPM", int(wpm))
		if int(wpm) > t.WPMTarget {
			rateStats = lipgloss.NewStyle().Foreground(Correct).Render(rateStats)
		} else {
			rateStats = lipgloss.NewStyle().Foreground(Incorrect).Render(rateStats)
		}

		total, correct, fixed, unfixed := 0, 0, 0, 0
		for _, keystroke := range t.Keystrokes {
			total += 1
			switch keystroke.Status {
			case CORRECT:
				correct += 1
			case ERROR_FIXED:
				fixed += 1
			case ERROR_UNFIXED:
				unfixed += 1
			}
		}

		charStats := lipgloss.JoinHorizontal(
			lipgloss.Center,
			fmt.Sprintf("%d", total), "/",
			lipgloss.NewStyle().Foreground(Correct).Render(fmt.Sprintf("%d", correct)), "/",
			lipgloss.NewStyle().Foreground(Fixed).Render(fmt.Sprintf("%d", fixed)), "/",
			lipgloss.NewStyle().Foreground(Incorrect).Render(fmt.Sprintf("%d", unfixed)),
		)

		accuracy := float32(correct) / float32(total)
		accuracyStats := fmt.Sprintf("%.02f%% Accuracy", accuracy*100.)

		res = lipgloss.JoinVertical(lipgloss.Center, res, rateStats, charStats, accuracyStats)
	}

	return res
}
