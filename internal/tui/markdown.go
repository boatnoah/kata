package tui

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ParseAssistantText splits an assistant response into blocks. Only fenced
// code blocks (``` on its own line, optional language identifier) are
// extracted; everything between fences becomes a single ProseBlock run.
// Headers, tables, blockquotes, and list bullets intentionally fall through
// as plain prose — the renderer paints the subset we care about (bold,
// italic, inline code) via styleInline at render time.
func ParseAssistantText(text string) []Block {
	if text == "" {
		return nil
	}
	lines := strings.Split(text, "\n")

	var blocks []Block
	var prose []string
	var code []string
	var lang string
	inFence := false

	flushProse := func() {
		if len(prose) == 0 {
			return
		}
		body := strings.Trim(strings.Join(prose, "\n"), "\n")
		if body != "" {
			blocks = append(blocks, ProseBlock{Text: body})
		}
		prose = prose[:0]
	}

	for _, line := range lines {
		trim := strings.TrimLeft(line, " \t")
		if strings.HasPrefix(trim, "```") {
			if !inFence {
				flushProse()
				lang = strings.TrimSpace(strings.TrimPrefix(trim, "```"))
				inFence = true
				continue
			}
			blocks = append(blocks, CodeBlock{Lang: lang, Body: strings.Join(code, "\n")})
			code = code[:0]
			lang = ""
			inFence = false
			continue
		}
		if inFence {
			code = append(code, line)
		} else {
			prose = append(prose, line)
		}
	}

	// Unclosed fence — emit what we have anyway so mid-stream code still
	// shows its gutter/highlight treatment.
	if inFence {
		blocks = append(blocks, CodeBlock{Lang: lang, Body: strings.Join(code, "\n")})
	}
	flushProse()
	return blocks
}

var (
	reInlineCode = regexp.MustCompile("`([^`\n]+?)`")
	reBold       = regexp.MustCompile(`\*\*([^*\n]+?)\*\*`)
	reItalic     = regexp.MustCompile(`\*([^*\n]+?)\*`)
	reLink       = regexp.MustCompile(`\[([^\]\n]+)\]\(([^)\s]+)\)`)
)

// styleInline applies light markdown inline styling: links, `code`, **bold**,
// and *italic*. Inline code and rendered links are stashed as placeholders
// before the bold/italic passes so those regexes can't touch the protected
// content (URLs often contain underscores and asterisks that would otherwise
// be misread as emphasis).
func styleInline(s string, theme Theme) string {
	codeStyle := lipgloss.NewStyle().Foreground(theme.Accent2)
	boldStyle := lipgloss.NewStyle().Bold(true)
	italicStyle := lipgloss.NewStyle().Italic(true)
	linkStyle := lipgloss.NewStyle().Foreground(theme.Accent2).Underline(true)

	var stashes []string
	stash := func(rendered string) string {
		stashes = append(stashes, rendered)
		return "\x00s" + strconv.Itoa(len(stashes)-1) + "\x00"
	}

	s = reLink.ReplaceAllStringFunc(s, func(m string) string {
		sub := reLink.FindStringSubmatch(m)
		label, url := sub[1], sub[2]
		return stash(hyperlink(url, linkStyle.Render(label)))
	})
	s = reInlineCode.ReplaceAllStringFunc(s, func(m string) string {
		return stash(codeStyle.Render(m[1 : len(m)-1]))
	})
	s = reBold.ReplaceAllStringFunc(s, func(m string) string {
		return boldStyle.Render(m[2 : len(m)-2])
	})
	s = reItalic.ReplaceAllStringFunc(s, func(m string) string {
		return italicStyle.Render(m[1 : len(m)-1])
	})
	for i, rendered := range stashes {
		s = strings.Replace(s, "\x00s"+strconv.Itoa(i)+"\x00", rendered, 1)
	}
	return s
}

// hyperlink wraps already-styled label text in an OSC 8 terminal hyperlink
// escape. Terminals that support OSC 8 (Ghostty, WezTerm, recent iTerm,
// Kitty, VS Code) make the label clickable; terminals that don't silently
// ignore the escape and show the styled label unchanged.
func hyperlink(url, label string) string {
	return "\x1b]8;;" + url + "\x1b\\" + label + "\x1b]8;;\x1b\\"
}
