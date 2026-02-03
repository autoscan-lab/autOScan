package settings

import (
	"fmt"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/feli05/autoscan/internal/config"
	"github.com/feli05/autoscan/internal/tui/components"
)

type State struct {
	Settings       *config.Settings
	SettingsCursor int
	Width          int
}

func View(s State) string {
	var b strings.Builder

	b.WriteString(components.RenderHeader("Settings"))

	boxWidth := components.BoxWidth(s.Width, 4, 80)
	box := components.BoxStyle(boxWidth)

	var content strings.Builder
	content.WriteString(components.SubtleText.Render("Display Options"))
	content.WriteString("\n\n")

	toggle1 := components.Toggle{
		Label:       "Short Names",
		Description: "Truncate folder names at first underscore",
		Value:       s.Settings.ShortNames,
		Focused:     s.SettingsCursor == 0,
	}
	content.WriteString(toggle1.View())
	content.WriteString("\n\n")

	toggle2 := components.Toggle{
		Label:       "Keep Binaries",
		Description: "Keep compiled binaries after grading",
		Value:       s.Settings.KeepBinaries,
		Focused:     s.SettingsCursor == 1,
	}
	content.WriteString(toggle2.View())
	content.WriteString("\n\n")

	cpuCount := runtime.NumCPU()
	var workersValue string
	if s.Settings.MaxWorkers == 0 {
		workersValue = fmt.Sprintf("All CPUs (%d)", cpuCount)
	} else {
		workersValue = fmt.Sprintf("%d (of %d CPUs)", s.Settings.MaxWorkers, cpuCount)
	}
	workersSetting := components.NumberSetting{
		Label:   "Max Workers",
		Value:   workersValue,
		Focused: s.SettingsCursor == 2,
		Description: []string{
			"Concurrent compilation processes",
			fmt.Sprintf("(0 = all %d CPUs, 2-4 for limited resources)", cpuCount),
		},
	}
	content.WriteString(workersSetting.View())

	content.WriteString("\n\n")
	content.WriteString(components.SubtleText.Render("Plagiarism Detection"))
	content.WriteString("\n\n")

	windowSetting := components.NumberSetting{
		Label:   "Window Size",
		Value:   fmt.Sprintf("%d", s.Settings.PlagiarismWindowSize),
		Focused: s.SettingsCursor == 3,
	}
	content.WriteString(windowSetting.View())
	content.WriteString("\n")

	minTokensSetting := components.NumberSetting{
		Label:   "Min Function Tokens",
		Value:   fmt.Sprintf("%d", s.Settings.PlagiarismMinFuncTokens),
		Focused: s.SettingsCursor == 4,
	}
	content.WriteString(minTokensSetting.View())
	content.WriteString("\n")

	thresholdSetting := components.NumberSetting{
		Label:   "Score Threshold",
		Value:   fmt.Sprintf("%.2f", s.Settings.PlagiarismScoreThreshold),
		Focused: s.SettingsCursor == 5,
	}
	content.WriteString(thresholdSetting.View())

	b.WriteString(box.Render(content.String()))

	b.WriteString("\n\n")
	b.WriteString(components.SubtleText.Render("  Config: ~/.config/autoscan/settings.yaml"))

	b.WriteString("\n\n")
	helpItems := []components.HelpItem{
		{Key: "↑/↓", Desc: "navigate"},
		{Key: "space/enter", Desc: "toggle"},
	}
	if s.SettingsCursor >= 2 {
		resetLabel := "reset"
		if s.SettingsCursor == 2 {
			resetLabel = "reset workers"
		} else {
			resetLabel = "reset default"
		}
		helpItems = append(helpItems, components.HelpItem{Key: "+/-", Desc: "adjust"}, components.HelpItem{Key: "0", Desc: resetLabel})
	}
	helpItems = append(helpItems, components.HelpItem{Key: "esc", Desc: "back"})
	b.WriteString(components.RenderHelpBar(helpItems))

	return b.String()
}

type UpdateResult struct {
	Settings       config.Settings
	SettingsCursor int
	GoBack         bool
}

func Update(s State, msg tea.KeyMsg) UpdateResult {
	result := UpdateResult{
		Settings:       *s.Settings,
		SettingsCursor: s.SettingsCursor,
		GoBack:         false,
	}

	switch msg.String() {
	case "j", "down":
		if result.SettingsCursor < 5 {
			result.SettingsCursor++
		}
	case "k", "up":
		if result.SettingsCursor > 0 {
			result.SettingsCursor--
		}
	case "enter", " ":
		switch result.SettingsCursor {
		case 0:
			result.Settings.ShortNames = !result.Settings.ShortNames
			config.SaveSettings(result.Settings)
		case 1:
			result.Settings.KeepBinaries = !result.Settings.KeepBinaries
			config.SaveSettings(result.Settings)
		}
	case "+", "=":
		switch result.SettingsCursor {
		case 2:
			if result.Settings.MaxWorkers < 32 {
				result.Settings.MaxWorkers++
				config.SaveSettings(result.Settings)
			}
		case 3:
			if result.Settings.PlagiarismWindowSize < 64 {
				result.Settings.PlagiarismWindowSize++
				config.SaveSettings(result.Settings)
			}
		case 4:
			if result.Settings.PlagiarismMinFuncTokens < 1024 {
				result.Settings.PlagiarismMinFuncTokens++
				config.SaveSettings(result.Settings)
			}
		case 5:
			if result.Settings.PlagiarismScoreThreshold < 1.0 {
				result.Settings.PlagiarismScoreThreshold += 0.05
				if result.Settings.PlagiarismScoreThreshold > 1.0 {
					result.Settings.PlagiarismScoreThreshold = 1.0
				}
				config.SaveSettings(result.Settings)
			}
		}
	case "-", "_":
		switch result.SettingsCursor {
		case 2:
			if result.Settings.MaxWorkers > 0 {
				result.Settings.MaxWorkers--
				config.SaveSettings(result.Settings)
			}
		case 3:
			if result.Settings.PlagiarismWindowSize > 1 {
				result.Settings.PlagiarismWindowSize--
				config.SaveSettings(result.Settings)
			}
		case 4:
			if result.Settings.PlagiarismMinFuncTokens > 1 {
				result.Settings.PlagiarismMinFuncTokens--
				config.SaveSettings(result.Settings)
			}
		case 5:
			if result.Settings.PlagiarismScoreThreshold > 0.0 {
				result.Settings.PlagiarismScoreThreshold -= 0.05
				if result.Settings.PlagiarismScoreThreshold < 0.0 {
					result.Settings.PlagiarismScoreThreshold = 0.0
				}
				config.SaveSettings(result.Settings)
			}
		}
	case "0":
		switch result.SettingsCursor {
		case 2:
			result.Settings.MaxWorkers = 0
			config.SaveSettings(result.Settings)
		case 3:
			result.Settings.PlagiarismWindowSize = 6
			config.SaveSettings(result.Settings)
		case 4:
			result.Settings.PlagiarismMinFuncTokens = 14
			config.SaveSettings(result.Settings)
		case 5:
			result.Settings.PlagiarismScoreThreshold = 0.6
			config.SaveSettings(result.Settings)
		}
	case "q", "esc":
		result.GoBack = true
	}

	return result
}
