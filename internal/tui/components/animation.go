package components

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/feli05/autoscan/internal/tui/styles"
)

type AnimationTickMsg time.Time

func AnimationTickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*400, func(t time.Time) tea.Msg {
		return AnimationTickMsg(t)
	})
}

type EyeAnimation struct {
	frame, maxFrames, width, height int
}

func NewEyeAnimation() EyeAnimation {
	return EyeAnimation{frame: 0, maxFrames: 16, width: 22, height: 6}
}

func (e *EyeAnimation) Update(msg tea.Msg) tea.Cmd {
	if _, ok := msg.(AnimationTickMsg); ok {
		e.frame = (e.frame + 1) % e.maxFrames
		return AnimationTickCmd()
	}
	return nil
}

func (e *EyeAnimation) Init() tea.Cmd { return AnimationTickCmd() }

func (e *EyeAnimation) View() string {
	frames := e.getFrames()
	if e.frame >= len(frames) {
		e.frame = 0
	}
	return frames[e.frame]
}

func (e *EyeAnimation) getFrames() []string {
	eyeStyle := styles.EyeColor
	pupilStyle := styles.EyePupilColor

	// Wide open eyes - center
	eyeOpen := []string{
		"  ╭────────╮  ",
		" ╱          ╲ ",
		"│    ◉  ◉    │",
		"│            │",
		" ╲          ╱ ",
		"  ╰────────╯  ",
	}

	// Eyes looking left
	eyeLookLeft := []string{
		"  ╭────────╮  ",
		" ╱          ╲ ",
		"│   ◉  ◉     │",
		"│            │",
		" ╲          ╱ ",
		"  ╰────────╯  ",
	}

	// Eyes looking right
	eyeLookRight := []string{
		"  ╭────────╮  ",
		" ╱          ╲ ",
		"│     ◉  ◉   │",
		"│            │",
		" ╲          ╱ ",
		"  ╰────────╯  ",
	}

	// Half closed (blinking)
	eyeHalf := []string{
		"              ",
		"  ╭────────╮  ",
		" ─  ━━  ━━  ─ ",
		"│            │",
		" ╲          ╱ ",
		"  ╰────────╯  ",
	}

	// Closed (blink)
	eyeClosed := []string{
		"              ",
		"  ╭────────╮  ",
		" ─ ──────── ─ ",
		" ╲          ╱ ",
		"  ╰────────╯  ",
		"              ",
	}

	// Eyes looking up
	eyeLookUp := []string{
		"  ╭────────╮  ",
		" ╱   ◉  ◉   ╲ ",
		"│            │",
		"│            │",
		" ╲          ╱ ",
		"  ╰────────╯  ",
	}

	// Build styled frames
	buildFrame := func(lines []string) string {
		var b strings.Builder
		for _, line := range lines {
			// Apply eye style
			styled := eyeStyle.Render(line)
			// Highlight pupils
			styled = strings.ReplaceAll(styled, "◉", pupilStyle.Render("◉"))
			b.WriteString(styled)
			b.WriteString("\n")
		}
		return b.String()
	}

	// Animation sequence with more frames for variety
	return []string{
		buildFrame(eyeOpen),      // 0: open center
		buildFrame(eyeOpen),      // 1: hold
		buildFrame(eyeOpen),      // 2: hold
		buildFrame(eyeLookLeft),  // 3: look left
		buildFrame(eyeLookLeft),  // 4: hold left
		buildFrame(eyeOpen),      // 5: center
		buildFrame(eyeOpen),      // 6: hold
		buildFrame(eyeLookRight), // 7: look right
		buildFrame(eyeLookRight), // 8: hold right
		buildFrame(eyeOpen),      // 9: center
		buildFrame(eyeHalf),      // 10: half close
		buildFrame(eyeClosed),    // 11: blink
		buildFrame(eyeOpen),      // 12: open
		buildFrame(eyeOpen),      // 13: hold
		buildFrame(eyeLookUp),    // 14: look up
		buildFrame(eyeOpen),      // 15: center
	}
}

