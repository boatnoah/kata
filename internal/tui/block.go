package tui

import (
	"path/filepath"
	"strconv"
	"strings"

	"github.com/boatnoah/kata/internal/agent"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// Block is a self-contained renderable unit. A single TranscriptItem may
// produce one or many blocks (an assistant response with mixed prose and
// fenced code becomes several). Each Block owns its own width math,
// styling, and line layout — the history pane just stitches the output
// together and manages the outer line cache.
type Block interface {
	Render(width int, theme Theme) []string
}

// ProseBlock is a run of plain assistant prose. Inline markdown (`code`,
// **bold**, *italic*) is resolved at render time via styleInline; block-
// level markdown constructs (headers, lists, blockquotes) are intentionally
// left as literal text — that's what "minimal" means here.
type ProseBlock struct {
	Text string
}

func (b ProseBlock) Render(width int, theme Theme) []string {
	body := styleInline(b.Text, theme)
	body = wrapToWidth(body, width)
	if body == "" {
		return nil
	}
	return strings.Split(body, "\n")
}

// CodeBlock is a fenced code block with optional language tag. Rendering
// delegates to highlightCode which paints the gutter, muted language label,
// and Chroma-based syntax highlighting.
type CodeBlock struct {
	Lang string
	Body string
}

func (b CodeBlock) Render(width int, theme Theme) []string {
	if b.Body == "" {
		return nil
	}
	return highlightCode(b.Body, b.Lang, theme, width)
}

// ThinkingBlock renders the model's pre-response thinking as a labelled,
// indented, muted-italic block. The leading "· Thinking" line gives the
// content a recognizable frame without leaning on a full bordered box.
type ThinkingBlock struct {
	Text string
}

func (b ThinkingBlock) Render(width int, theme Theme) []string {
	// When there is no thinking content, avoid rendering a "Thinking" header.
	// The history view will still show the spinner/status line for the item.
	if strings.TrimSpace(b.Text) == "" {
		return nil
	}

	labelStyle := lipgloss.NewStyle().Foreground(theme.FgDim)
	bodyStyle := lipgloss.NewStyle().Foreground(theme.FgDim).Italic(true)

	out := []string{labelStyle.Render("· Thinking")}
	bodyWidth := width - 2
	if bodyWidth < 1 {
		bodyWidth = width
	}
	for _, line := range strings.Split(wrapToWidth(b.Text, bodyWidth), "\n") {
		out = append(out, "  "+bodyStyle.Render(line))
	}
	return out
}

// ToolRole positions a ToolBlock inside a run of adjacent same-verb tool
// calls. A group of consecutive "Read foo.go", "Read bar.go", "Read baz.go"
// events renders as a single `⏺ Read` header followed by `  ⎿ <file>` lines,
// matching Claude Code's batched tool display instead of three separate
// glyph lines.
type ToolRole int

const (
	ToolRoleSingle ToolRole = iota
	ToolRoleGroupStart
	ToolRoleContinuation
)

// ToolBlock renders a tool call. Layout depends on Role:
//   - ToolRoleSingle: one line, `⏺ <verb bold> <arg muted>`, with an
//     optional `  └ <detail>` below when Detail is set.
//   - ToolRoleGroupStart: `⏺ <verb bold>` header, then `  ⎿ <arg>` for this
//     item's argument.
//   - ToolRoleContinuation: `  ⎿ <arg>` only — no glyph, no verb.
type ToolBlock struct {
	Title  string
	Detail string
	State  ToolState
	Role   ToolRole
}

func (b ToolBlock) Render(width int, theme Theme) []string {
	glyphColor := theme.FgDim
	switch b.State {
	case ToolOK:
		glyphColor = theme.Green
	case ToolErr:
		glyphColor = theme.Red
	}
	glyph := lipgloss.NewStyle().Foreground(glyphColor).Render("⏺")

	verbStyle := lipgloss.NewStyle().Foreground(theme.Fg).Bold(true)
	argStyle := lipgloss.NewStyle().Foreground(theme.FgDim)
	leaderStyle := lipgloss.NewStyle().Foreground(theme.FgDim)
	verb, arg := splitTitle(b.Title)

	contWidth := width - 4
	if contWidth < 1 {
		contWidth = width
	}
	continuationLine := func(text string) string {
		return "  " + leaderStyle.Render("⎿ ") + ansi.Truncate(argStyle.Render(text), contWidth, "…")
	}

	switch b.Role {
	case ToolRoleContinuation:
		if arg == "" {
			return nil
		}
		return []string{continuationLine(arg)}
	case ToolRoleGroupStart:
		header := ansi.Truncate(glyph+" "+verbStyle.Render(verb), width, "…")
		if arg == "" {
			return []string{header}
		}
		return []string{header, continuationLine(arg)}
	}

	titleLine := verbStyle.Render(verb)
	if arg != "" {
		titleLine += " " + argStyle.Render(arg)
	}
	out := []string{ansi.Truncate(glyph+" "+titleLine, width, "…")}

	if strings.TrimSpace(b.Detail) == "" {
		return out
	}
	detailWidth := width - 4
	if detailWidth < 1 {
		detailWidth = width
	}
	out = append(out, "  "+leaderStyle.Render("└ ")+ansi.Truncate(argStyle.Render(b.Detail), detailWidth, "…"))
	return out
}

// splitTitle separates the first token (the verb — "Read", "Ran", "Searched")
// from the rest of the title so ToolBlock can render them with different
// weights. Single-word titles ("Working") leave arg empty.
func splitTitle(title string) (string, string) {
	title = strings.TrimSpace(title)
	if idx := strings.IndexByte(title, ' '); idx > 0 {
		return title[:idx], strings.TrimSpace(title[idx+1:])
	}
	return title, ""
}

// UserBlock renders a user turn with the "❯" prompt glyph. Wrapping leaves
// room for the 2-char prefix; continuation lines indent to align under the
// first body character.
type UserBlock struct {
	Text string
}

func (b UserBlock) Render(width int, theme Theme) []string {
	prompt := lipgloss.NewStyle().Foreground(theme.Accent).Bold(true).Render("❯")
	textStyle := lipgloss.NewStyle().Foreground(theme.FgBright).Bold(true)

	bodyWidth := width - 2
	if bodyWidth < 8 {
		bodyWidth = 8
	}
	lines := strings.Split(wrapToWidth(b.Text, bodyWidth), "\n")
	for i, line := range lines {
		lines[i] = textStyle.Render(line)
	}
	lines[0] = prompt + " " + lines[0]
	for i := 1; i < len(lines); i++ {
		lines[i] = "  " + lines[i]
	}
	return lines
}

// ErrorBlock renders an error in the theme's red, wrapped to width.
type ErrorBlock struct {
	Text string
}

func (b ErrorBlock) Render(width int, theme Theme) []string {
	style := lipgloss.NewStyle().Foreground(theme.Red)
	lines := strings.Split(wrapToWidth(b.Text, width), "\n")
	for i, line := range lines {
		lines[i] = style.Render(line)
	}
	return lines
}

// DiffBlock renders one file's changes as a Cursor-style inline diff card:
// a path+stat header, syntax-highlighted hunks with a colored sign column,
// and a truncation footer when MaxLines is set.
type DiffBlock struct {
	File     agent.FileChange
	MaxLines int // 0 = no truncation
}

func (b DiffBlock) Render(width int, theme Theme) []string {
	if b.File.Path == "" && len(b.File.Hunks) == 0 {
		return nil
	}
	lines := []string{diffHeader(b.File, width, theme)}
	lines = append(lines, diffBody(b.File, width, theme, b.MaxLines)...)
	return lines
}

func diffHeader(fc agent.FileChange, width int, theme Theme) string {
	pathStyle := lipgloss.NewStyle().Foreground(theme.FgBright).Bold(true)
	statAdd := lipgloss.NewStyle().Foreground(theme.Green)
	statDel := lipgloss.NewStyle().Foreground(theme.Red)
	tagStyle := lipgloss.NewStyle().Foreground(theme.FgDim)

	display := displayPath(fc.Path)
	if fc.Op == agent.FileChangeUpdate && fc.MovePath != "" {
		display += " → " + displayPath(fc.MovePath)
	}

	var tag string
	switch fc.Op {
	case agent.FileChangeAdd:
		tag = "new"
	case agent.FileChangeDelete:
		tag = "deleted"
	}

	parts := []string{pathStyle.Render(display)}
	if tag != "" {
		parts = append(parts, tagStyle.Render(tag))
	}
	if fc.Added > 0 {
		parts = append(parts, statAdd.Render("+"+strconv.Itoa(fc.Added)))
	}
	if fc.Removed > 0 {
		parts = append(parts, statDel.Render("-"+strconv.Itoa(fc.Removed)))
	}
	return ansi.Truncate(strings.Join(parts, " "), width, "…")
}

func diffBody(fc agent.FileChange, width int, theme Theme, maxLines int) []string {
	signWidth := 2 // "+ ", "- ", "  "
	codeWidth := width - signWidth
	if codeWidth < 1 {
		codeWidth = 1
	}

	var body []string
	lang := langFromPath(fc.Path)
	for _, hunk := range fc.Hunks {
		if hunk.Header != "" {
			body = append(body, lipgloss.NewStyle().Foreground(theme.FgDim).Italic(true).Render(ansi.Truncate(hunk.Header, width, "…")))
		}
		for _, line := range hunk.Lines {
			body = append(body, renderDiffLine(line, lang, theme, codeWidth))
		}
	}

	if maxLines > 0 && len(body) > maxLines {
		keep := maxLines - 1
		if keep < 1 {
			keep = 1
		}
		hidden := len(body) - keep
		body = append(body[:keep], lipgloss.NewStyle().Foreground(theme.FgDim).Render("… truncated ("+strconv.Itoa(hidden)+" more lines) · :diff to review"))
	}
	return body
}

func renderDiffLine(line agent.DiffLine, lang string, theme Theme, codeWidth int) string {
	var signColor lipgloss.Color
	var sign string
	switch line.Kind {
	case agent.DiffLineAdd:
		signColor, sign = theme.Green, "+"
	case agent.DiffLineRemove:
		signColor, sign = theme.Red, "-"
	default:
		signColor, sign = theme.FgDim, " "
	}
	signStyle := lipgloss.NewStyle().Foreground(signColor).Bold(true)
	prefix := signStyle.Render(sign) + " "

	var code string
	if line.Kind == agent.DiffLineContext {
		code = lipgloss.NewStyle().Foreground(theme.FgDim).Render(line.Text)
	} else {
		// Syntax-highlight the code portion; the sign column keeps the +/-
		// signal visible independent of the highlighter's colors.
		code = strings.TrimRight(chromaHighlight(line.Text, lang, theme), "\n")
	}
	return prefix + ansi.Truncate(code, codeWidth, "…")
}

func displayPath(p string) string {
	return p
}

// langFromPath picks a Chroma lexer hint from a file path's extension.
// Unknown extensions return the empty string (Chroma falls back to a
// plain-text lexer).
func langFromPath(p string) string {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(p)), ".")
	switch ext {
	case "go":
		return "go"
	case "ts", "tsx":
		return "typescript"
	case "js", "jsx":
		return "javascript"
	case "py":
		return "python"
	case "rs":
		return "rust"
	case "md":
		return "markdown"
	case "json":
		return "json"
	case "yaml", "yml":
		return "yaml"
	case "sh", "bash":
		return "bash"
	case "toml":
		return "toml"
	}
	return ""
}

// SystemBlock renders a muted system note wrapped to width.
type SystemBlock struct {
	Text string
}

func (b SystemBlock) Render(width int, theme Theme) []string {
	style := lipgloss.NewStyle().Foreground(theme.FgDim)
	lines := strings.Split(wrapToWidth(b.Text, width), "\n")
	for i, line := range lines {
		lines[i] = style.Render(line)
	}
	return lines
}
