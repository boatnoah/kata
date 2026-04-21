package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// chrome owns the topbar/statusline rendering — the thin bands that wrap
// the chat + prompt. The actual session/context values come from the App
// via a small snapshot so this stays pure presentation.

type chromeSnapshot struct {
	theme           Theme
	width           int
	path            string
	title           string
	sessionID       string
	mode            Mode
	scope           string
	branch          string
	provider        string
	model           string
	ctxUsed         int
	ctxTotal        int
	msgCount        int
	filesMod        int
	notice          string
	lineCol         string
	pendingApproval bool
}

// renderTopbar produces the 1-row header: `kata · <path> · <title>   session xxxx`.
func renderTopbar(s chromeSnapshot) string {
	if s.width <= 0 {
		return ""
	}
	t := s.theme
	brand := lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render("kata")
	sep := lipgloss.NewStyle().Foreground(t.FgDimmer).Render("·")
	path := lipgloss.NewStyle().Foreground(t.FgDim).Render(s.path)

	left := brand + " " + sep + " " + path
	if s.title != "" {
		title := lipgloss.NewStyle().Foreground(t.FgDimmer).Render(s.title)
		left = left + " " + sep + " " + title
	}

	right := ""
	if s.sessionID != "" {
		right = lipgloss.NewStyle().Foreground(t.FgDimmer).Render("session " + s.sessionID)
	}

	return padBetween(" "+left, right+" ", s.width, t.FgDimmer)
}

// renderStatusline produces the 1-row footer: mode segment + branch/model/ctx/line:col.
func renderStatusline(s chromeSnapshot) string {
	if s.width <= 0 {
		return ""
	}
	t := s.theme
	modeColor := t.ModeColor(s.mode)

	// Mode segment: painted pill-style band with scope ▸ MODE.
	scopeText := fmt.Sprintf(" %s %s %s ", s.scope, "▸", modeLabel(s.mode))
	modeSeg := lipgloss.NewStyle().
		Foreground(t.Bg).
		Background(modeColor).
		Bold(true).
		Render(scopeText)

	branch := lipgloss.NewStyle().Foreground(t.FgDim).Render(" " + s.branch)
	middle := ""
	if s.notice != "" {
		middle = lipgloss.NewStyle().Foreground(t.Accent).Render(s.notice)
	} else {
		parts := []string{}
		if s.msgCount > 0 {
			parts = append(parts, fmt.Sprintf("%d msgs", s.msgCount))
		}
		if s.filesMod > 0 {
			parts = append(parts, fmt.Sprintf("%d files modified", s.filesMod))
		}
		if s.pendingApproval {
			parts = append(parts, "approval pending")
		}
		middle = lipgloss.NewStyle().Foreground(t.FgDim).Render(strings.Join(parts, " · "))
	}

	rightSegs := []string{}
	if label := modelLabel(s.provider, s.model); label != "" {
		rightSegs = append(rightSegs, lipgloss.NewStyle().Foreground(t.FgDim).Render(label))
	}
	if s.ctxTotal > 0 {
		rightSegs = append(rightSegs, lipgloss.NewStyle().Foreground(t.FgDim).
			Render(fmt.Sprintf("ctx %s / %s", humanTokens(s.ctxUsed), humanTokens(s.ctxTotal))))
	}
	if s.lineCol != "" {
		rightSegs = append(rightSegs, lipgloss.NewStyle().Foreground(t.FgDim).Render(s.lineCol))
	}
	rightJoin := strings.Join(rightSegs, "  ")
	right := ""
	if rightJoin != "" {
		right = rightJoin + " "
	}

	left := modeSeg + "  " + branch
	if middle != "" {
		left = left + "  " + middle
	}

	return padBetween(left, right, s.width, t.Border)
}

// padBetween renders left flush-left and right flush-right, filling the
// middle with spaces, clamped to the given width. When combined width
// exceeds the pane, the right side is truncated to keep the left visible.
func padBetween(left, right string, width int, _ lipgloss.Color) string {
	lw := lipgloss.Width(left)
	rw := lipgloss.Width(right)
	if lw+rw >= width {
		// Truncate right if it doesn't fit.
		if lw >= width {
			return truncateToWidth(left, width)
		}
		right = truncateToWidth(right, width-lw)
		rw = lipgloss.Width(right)
	}
	gap := width - lw - rw
	if gap < 0 {
		gap = 0
	}
	return left + strings.Repeat(" ", gap) + right
}

// modelLabel formats the statusline's AI badge as "provider · model" when
// both are known, falling back to whichever half is populated. Keeping this
// in one place means adding a second provider later only changes the
// upstream source, not every render site.
func modelLabel(provider, model string) string {
	switch {
	case provider != "" && model != "":
		return provider + " · " + model
	case model != "":
		return model
	default:
		return provider
	}
}

func modeLabel(m Mode) string {
	switch m {
	case ModeInsert:
		return "INSERT"
	case ModeVisual:
		return "VISUAL"
	case ModeCommandLine:
		return "COMMAND"
	default:
		return "NORMAL"
	}
}

// humanTokens formats a token count as e.g. "74.8k" / "200k" / "812".
func humanTokens(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 100_000 {
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	}
	return fmt.Sprintf("%dk", n/1000)
}

// shortPath collapses $HOME to `~` so the topbar stays tight on wide cwds.
func shortPath(p string) string {
	if p == "" {
		return ""
	}
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		if rel, err := filepath.Rel(home, p); err == nil && !strings.HasPrefix(rel, "..") {
			if rel == "." {
				return "~"
			}
			return "~/" + rel
		}
	}
	return p
}
