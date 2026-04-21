package tui

import (
	"bytes"
	"strings"

	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// highlightCode returns the code body rendered as gutter-prefixed, syntax-
// highlighted lines. When lang is non-empty a muted label is emitted first
// (no gutter). Long lines truncate with an ellipsis — code blocks never
// word-wrap.
func highlightCode(body, lang string, theme Theme, width int) []string {
	gutter := lipgloss.NewStyle().Foreground(theme.Accent).Render("│ ")
	codeWidth := width - lipgloss.Width(gutter)
	if codeWidth < 1 {
		codeWidth = 1
	}

	var out []string
	if lang != "" {
		out = append(out, lipgloss.NewStyle().Foreground(theme.FgDim).Render(lang))
	}

	highlighted := chromaHighlight(body, lang, theme)
	for _, line := range strings.Split(strings.TrimRight(highlighted, "\n"), "\n") {
		out = append(out, gutter+ansi.Truncate(line, codeWidth, "…"))
	}
	return out
}

// chromaStyleFor maps our TUI theme to the Chroma syntax style that looks
// most at home against that palette. Unknown names fall back to monokai.
func chromaStyleFor(theme Theme) string {
	switch theme.Name {
	case "gruvbox":
		return "gruvbox"
	case "nord":
		return "nord"
	case "solarized":
		return "solarized-dark256"
	case "paper":
		return "monokailight"
	case "mono":
		return "bw"
	default:
		return "monokai"
	}
}

func chromaHighlight(body, lang string, theme Theme) string {
	lexer := lexers.Get(lang)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	style := styles.Get(chromaStyleFor(theme))
	if style == nil {
		style = styles.Fallback
	}
	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Fallback
	}
	iterator, err := lexer.Tokenise(nil, body)
	if err != nil {
		return body
	}
	var buf bytes.Buffer
	if err := formatter.Format(&buf, style, iterator); err != nil {
		return body
	}
	return buf.String()
}
