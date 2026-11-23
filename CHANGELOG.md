# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [v0.4.1] - 2026-11-23

### Fixed

- A partial theme with `InlineCodeBG: ""` no longer picks up the default `#3a3a3a` highlight background. Empty now means transparent again, so code spans keep only the caller's foreground color. Set `InlineCodeBG` to `"default"` to inherit the terminal background, or to a color to restore a highlight. The zero `Theme` still uses every `DefaultTheme()` color, including the highlight.

## [v0.4.0] - 2026-11-23

### Changed

- Tables now paint one blank line between body rows by default; pass `CompactTableRows` as `Options.TableRowSpacing` for the dense layout. Consumers that count painted rows must account for these spacer lines (`RenderLines` includes them).
- Partial themes now inherit unspecified colors from `DefaultTheme()` instead of blanking them. A caller that deliberately leaves a field empty for transparency (for example `Theme{InlineCodeBG: ""}`) must now set `"default"` explicitly, or use `NoColorTheme()`.

### Added

- `tviewmd.PlainText([]Segment) string` for extracting plain text from a segment slice, and `tviewmd.RenderLines(src, opts) []string` for the planned physical lines.
- `ErrStreamTooLarge` and `DefaultMaxStreamBytes` for the `Renderer` stream cap.
- Configurable table row spacing via `Options.TableRowSpacing` and `CompactTableRows`; tables paint one blank line between body rows by default.
- Per-field theme defaulting with `Theme.WithDefaults`, plus `NoColorTheme` for NO_COLOR / monochrome terminals.
- `Renderer.SetOptions`, `Renderer.SetWidth`, `Renderer.Err` and `Options.MaxStreamBytes` for reconfiguring a live stream and capping runaway input.
- `Block.Blocks` and `ListItem.Blocks` preserve nested code fences, tables, lists, headings and multiple paragraphs inside blockquotes and list items.

### Fixed

- Sanitize ESC/CSI/OSC and C0/C1 control characters at parse and paint time so streamed text cannot inject tview tags or terminal escapes.
- Preserve emphasis on links and inline code spans instead of dropping it.
- Adjacent links sharing a URL keep separate decorations (`Segment.LinkID`), in both the unwrapped and wrapped table paths.
- Parsing nested blockquotes/list items is linear again; parsing both the flattened and block views of the same children had made it exponential (a few dozen nesting levels could hang a render).
- Quoted lists resume the quote style after inline links, matching direct quote paragraphs.
- Hand-built `Block` tables measure exactly what they paint; untrusted strings are sanitized once at the render boundary.
- A bare `CR` is normalized to a line break instead of being deleted, so CR-only input no longer joins lines.
- Color values from a caller-supplied `Theme` can no longer inject tview tags.
- Empty fenced code blocks no longer emit tag-only blank output.
- Documentation and CI consistency: golangci-lint v2 config, install script and CI aligned; the CI Go matrix now matches `go.mod`; releases no longer attach a fake binary built from the library root.

## [v0.3.1] - 2026-10-20

### Fixed

- Paint one link decoration per link run and resume the ambient block style after inner tag resets.
- Derive the table column floor from painted width; share the column planner and styled-run branches.
- Preview CLI rejects non-positive widths and skips wide-glyph continuation cells.
- Correct the tview event-loop guidance in the streaming docs.
- Run the fuzz target in its package instead of `./...`.
- Satisfy golangci-lint v2 on the preview CLI and render path.

[Unreleased]: https://github.com/buchenberg/tviewmd/compare/v0.4.1...HEAD
[v0.4.1]: https://github.com/buchenberg/tviewmd/releases/tag/v0.4.1
[v0.4.0]: https://github.com/buchenberg/tviewmd/releases/tag/v0.4.0
[v0.3.1]: https://github.com/buchenberg/tviewmd/releases/tag/v0.3.1
