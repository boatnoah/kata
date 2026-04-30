# Kata

Vim-centric AI coding assistant for fast terminal workflows.

[![CI](https://github.com/boatnoah/kata/actions/workflows/ci.yml/badge.svg)](https://github.com/boatnoah/kata/actions/workflows/ci.yml)

> **Status:** early/experimental. Expect breaking changes.

<!--
Demo (placeholder):
Add a short GIF showing a full loop: open Kata → propose patch → approve → apply.
![Kata demo](docs/assets/demo.gif)
-->

## What is it?

`kata` is a terminal UI (TUI) that keeps you in a **Vim-like flow** while you:

- draft changes
- review diffs
- approve/apply patches
- iterate quickly without leaving the terminal

*(Feature list evolving — placeholders below until the UX settles.)*

## Quickstart

Launch the TUI:

```bash
kata
```

## Installation

### Homebrew (placeholder)

```bash
brew install boatnoah/tap/kata
```

### Go install

```bash
go install github.com/boatnoah/kata/cmd/kata@main
```

### Releases (placeholder)

Download a binary from GitHub Releases:
`https://github.com/boatnoah/kata/releases`

## Usage

### Typical workflow (placeholder)

- Start `kata`
- Connect to your model/provider *(TBD)*
- Ask for a change
- Review the diff
- Approve to apply

## Keybindings (Vim-ish)

These are the interactions currently tested in `internal/tui`:

### Navigation

- `h` `j` `k` `l`: move cursor
- `gg` / `G`: jump to top / bottom *(placeholder if not implemented yet)*

### Editing

- `i`: enter insert mode
- `a`: append
- `I`: insert at first non-space
- `A`: append at end of line
- `o`: open a line below
- `O`: open a line above
- `dd`: delete line
- `D`: delete line (single-key variant)

### Visual / selection

- `v`: visual mode *(placeholder if partial)*
- `y`: yank selection
- `p`: paste
- `Esc`: return to normal mode / cancel selection

> If you notice a mismatch between docs and behavior, please open an issue — the
> UI is moving quickly.

## Configuration (placeholder)

Configuration file format and location are not finalized yet.

- **Config path**: `~/.config/kata/config.toml` *(placeholder)*
- **Example**:

```toml
# placeholder
provider = "..."
model = "..."
```

## Development

Run locally:

```bash
go run ./cmd/kata
```

Run tests:

```bash
go test ./...
```

## Contributing

- Use **Conventional Commits** for PR titles (enforced in CI).
- Keep PR titles **≤ 72 characters** (also enforced).
- Fill out the PR template sections: **Summary** + **Test plan**.

## License (placeholder)

Add a `LICENSE` file and update this section.

## Feedback

If you’re using `kata`, I’d love feedback:

- GitHub Issues: `https://github.com/boatnoah/kata/issues/new`
