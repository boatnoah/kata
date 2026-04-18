package tui

import "github.com/charmbracelet/lipgloss"

// Theme is the active color palette. It mirrors the CSS variables the
// design prototype uses (bg/fg tiers + accents + per-mode colors) so all
// surfaces can derive styles from the same source.
type Theme struct {
	Name string

	Bg       lipgloss.Color
	Bg1      lipgloss.Color
	Bg2      lipgloss.Color
	Fg       lipgloss.Color
	FgDim    lipgloss.Color
	FgDimmer lipgloss.Color
	FgBright lipgloss.Color
	Border   lipgloss.Color
	BorderHi lipgloss.Color

	Accent  lipgloss.Color // cyan-ish primary accent (NORMAL mode)
	Accent2 lipgloss.Color // amber-ish secondary accent (VISUAL mode, thinking)
	Green   lipgloss.Color // success/additions (INSERT mode)
	Red     lipgloss.Color // errors/deletions
	Magenta lipgloss.Color // command-mode accent
	Blue    lipgloss.Color // misc.
}

func (t Theme) ModeColor(m Mode) lipgloss.Color {
	switch m {
	case ModeInsert:
		return t.Green
	case ModeVisual:
		return t.Accent2
	case ModeCommandLine:
		return t.Magenta
	default:
		return t.Accent
	}
}

var themes = map[string]Theme{
	"slate": {
		Name:     "slate",
		Bg:       "#0d1015",
		Bg1:      "#141921",
		Bg2:      "#1b2028",
		Fg:       "#c9d1d9",
		FgDim:    "#7d8590",
		FgDimmer: "#545d68",
		FgBright: "#e6edf3",
		Border:   "#1f252f",
		BorderHi: "#2a313c",
		Accent:   "#7ec7d6",
		Accent2:  "#d6b87e",
		Green:    "#88b888",
		Red:      "#c47f7f",
		Magenta:  "#b088c4",
		Blue:     "#7ea0d6",
	},
	"gruvbox": {
		Name:     "gruvbox",
		Bg:       "#1d2021",
		Bg1:      "#282828",
		Bg2:      "#32302f",
		Fg:       "#ebdbb2",
		FgDim:    "#a89984",
		FgDimmer: "#7c6f64",
		FgBright: "#fbf1c7",
		Border:   "#32302f",
		BorderHi: "#504945",
		Accent:   "#83a598",
		Accent2:  "#d79921",
		Green:    "#b8bb26",
		Red:      "#fb4934",
		Magenta:  "#d3869b",
		Blue:     "#83a598",
	},
	"nord": {
		Name:     "nord",
		Bg:       "#2e3440",
		Bg1:      "#353b48",
		Bg2:      "#3b4252",
		Fg:       "#d8dee9",
		FgDim:    "#8f98a8",
		FgDimmer: "#6c7a94",
		FgBright: "#eceff4",
		Border:   "#353b48",
		BorderHi: "#4c566a",
		Accent:   "#88c0d0",
		Accent2:  "#ebcb8b",
		Green:    "#a3be8c",
		Red:      "#bf616a",
		Magenta:  "#b48ead",
		Blue:     "#81a1c1",
	},
	"mono": {
		Name:     "mono",
		Bg:       "#0a0a0a",
		Bg1:      "#121212",
		Bg2:      "#1a1a1a",
		Fg:       "#d4d4d4",
		FgDim:    "#808080",
		FgDimmer: "#555555",
		FgBright: "#f5f5f5",
		Border:   "#1a1a1a",
		BorderHi: "#2a2a2a",
		Accent:   "#d4d4d4",
		Accent2:  "#bdbdbd",
		Green:    "#c8c8c8",
		Red:      "#c8c8c8",
		Magenta:  "#c8c8c8",
		Blue:     "#c8c8c8",
	},
	"solarized": {
		Name:     "solarized",
		Bg:       "#002b36",
		Bg1:      "#073642",
		Bg2:      "#0b4250",
		Fg:       "#93a1a1",
		FgDim:    "#657b83",
		FgDimmer: "#586e75",
		FgBright: "#eee8d5",
		Border:   "#0b4250",
		BorderHi: "#163f4d",
		Accent:   "#2aa198",
		Accent2:  "#b58900",
		Green:    "#859900",
		Red:      "#dc322f",
		Magenta:  "#d33682",
		Blue:     "#268bd2",
	},
	"paper": {
		Name:     "paper",
		Bg:       "#f5f1e8",
		Bg1:      "#ece7dc",
		Bg2:      "#e3ded0",
		Fg:       "#3b3a36",
		FgDim:    "#75716a",
		FgDimmer: "#a29d93",
		FgBright: "#111111",
		Border:   "#d9d3c5",
		BorderHi: "#b9b3a4",
		Accent:   "#3b7a82",
		Accent2:  "#a87830",
		Green:    "#5a7d3a",
		Red:      "#9c3a3a",
		Magenta:  "#8a4a78",
		Blue:     "#3b5a82",
	},
}

// ThemeNames returns the list of supported scheme names in canonical order.
func ThemeNames() []string {
	return []string{"slate", "gruvbox", "nord", "mono", "solarized", "paper"}
}

// ThemeByName resolves a theme name; returns the slate theme and false if
// the name is unknown.
func ThemeByName(name string) (Theme, bool) {
	t, ok := themes[name]
	if !ok {
		return themes["slate"], false
	}
	return t, true
}

// DefaultTheme is the initial theme used on startup.
func DefaultTheme() Theme { return themes["slate"] }
