// Command preview renders sample markdown tables through the real
// tview.TextView pipeline into a tcell SimulationScreen and dumps the painted
// cells as plain text. It is a look-at-it tool, not a test: no assertions.
//
// Usage:
//
//	go run ./cmd/preview [width] [name-filter]
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/buchenberg/tviewmd"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func main() {
	width, filter := parseArgs(os.Args[1:])

	shown := 0
	for _, s := range samples() {
		if filter != "" && !strings.Contains(s.name, filter) {
			continue
		}
		shown++
		fmt.Printf("\n=== %s === width %d\n\n", s.name, width)
		for _, line := range paint(s.src, width, 80) {
			fmt.Printf("%s\n", line)
		}
	}
	if shown == 0 {
		fmt.Fprintf(os.Stderr, "no sample matched %q\n", filter)
		os.Exit(1)
	}
}

// defaultWidth is the paint width used when no numeric argument is given.
const defaultWidth = 100

// parseArgs interprets the command line: an optional leading width, then an
// optional sample-name filter. A leading argument that is not an integer is the
// filter, so `go run ./cmd/preview docs` filters to "docs" instead of the
// argument being silently dropped; a width can still be given first
// (`go run ./cmd/preview 120 docs`).
func parseArgs(args []string) (width int, filter string) {
	width = defaultWidth
	if len(args) > 0 {
		if n, err := strconv.Atoi(args[0]); err == nil {
			width = n
			args = args[1:]
		}
	}
	if len(args) > 0 {
		filter = args[0]
	}
	return width, filter
}

// paint renders src the way the TUI does — tagged text into a wrapping
// TextView — and returns the painted, right-trimmed screen lines.
func paint(src string, w, h int) []string {
	tv := tview.NewTextView().SetDynamicColors(true).SetWrap(true)
	tv.SetText(tviewmd.Render(src, tviewmd.Options{Width: w}))
	tv.SetRect(0, 0, w, h)

	screen := tcell.NewSimulationScreen("")
	if err := screen.Init(); err != nil {
		panic(err)
	}
	defer screen.Fini()
	screen.SetSize(w, h)
	tv.Draw(screen)
	screen.Show()

	cells, sw, _ := screen.GetContents()
	rows := make([]string, h)
	for row := 0; row < h; row++ {
		var b strings.Builder
		for col := 0; col < sw; col++ {
			c := cells[row*sw+col]
			if len(c.Bytes) == 0 {
				b.WriteByte(' ')
				continue
			}
			// A double-width glyph paints one bytes-cell plus a filler cell
			// that tcell never writes; the space below is its second column.
			b.Write(c.Bytes)
		}
		rows[row] = strings.TrimRight(b.String(), " ")
	}
	for len(rows) > 0 && rows[len(rows)-1] == "" {
		rows = rows[:len(rows)-1]
	}
	return rows
}

type sample struct {
	name string
	src  string
}

func samples() []sample {
	return []sample{
		{"small", smallTable},
		{"wide", wideTable},
		{"prose", proseTable},
		{"links", linkTable},
		{"unicode", unicodeTable},
		{"long-token", longTokenTable},
		{"many-rows", manyRowsTable},
		{"document", document},
	}
}

const smallTable = `| Tool | Purpose | Built in |
|------|---------|:--------:|
| read | Read a file | yes |
| grep | Search contents | yes |
| edit | Exact string replace | yes |
`

const wideTable = `| Role | Provider | Model | Turns | Tools | Tokens | $ cost | Status |
|------|----------|-------|------:|------:|-------:|-------:|--------|
| golang-developer | zai | GLM-5.3-Flash | 12 | 8 | 41,208 | 0.021 | ok |
| reviewer | zai | GLM-5.3-Flash | 7 | 5 | 18,933 | 0.009 | ok |
| golang-tester | zai | GLM-5.3-Flash | 4 | 2 | 6,110 | 0.003 | failed |
| analyst | zai | GLM-5.3-Flash | 9 | 11 | 27,004 | 0.013 | ok |
| security_auditor | zai | GLM-5.3-Flash | 15 | 19 | 62,775 | 0.031 | timeout |
| counter | zai | GLM-5.3-Flash | 2 | 1 | 1,204 | 0.001 | ok |
`

const proseTable = `| Phase | What happens | What goes wrong |
|-------|--------------|-----------------|
| Dispatch | The orchestrator selects a role, builds the system prompt from the role file plus the environment header, and forks a worktree from the current HEAD so the sub-agent cannot disturb the parent tree while it works. | The fork point is the tree *at dispatch time*, so a long running sub-agent reviews code that has since moved; proposals then need a three-way merge instead of a fast-forward. |
| Work | The sub-agent runs its own agent loop with its own tool budget, iterating until it reports completion or exhausts its turns; every tool call is recorded in the trace store for later inspection. | A model that exhausts its output budget mid-reasoning emits finish_reason=length with zero content and zero tool calls, which looks like a clean finish to anything that does not check the reason explicitly. |
| Proposal | Changed files are sealed into a diff held to one side; nothing lands on the parent working tree until the orchestrator reads the diff and settles it. | Two proposals touching the same path can only ever be selected, never applied on top of each other, because the merge is path-disjoint by construction. |
| Settlement | Select fast-forwards when the parent has not moved; apply three-way merges when it has. Both fail closed rather than leaving a half-applied tree. | A settlement that races with an editor save on the same file can fail closed, and the work is then only recoverable by re-dispatching the whole task. |
`

const linkTable = `| Document | Summary | Path |
|----------|---------|------|
| [architecture](docs/architecture.md) | Runtime design: the loop, the event seam, and how consumers attach | docs/architecture.md |
| [sub-agents](docs/sub-agents.md) | Roles, budgets, the retained lane and proposal settlement | docs/sub-agents.md |
| [configuration](docs/configuration.md) | Config keys, env substitution, and validation rules | docs/configuration.md |
| [features](docs/features.md) | What ships in the binary, and what deliberately does not | docs/features.md |
`

const unicodeTable = `| ロール | 説明 | 状態 | Emoji |
|--------|------|------|-------|
| 開発者 | Go の機能実装とリファクタリングを担当します | 稼働中 | 🟢 |
| レビュアー | コードを検査し、複雑度を測定し、所見を報告します | 待機中 | ⏳ |
| テスター | テストスイートを実行し、失敗を分析します | 失敗 | 🌀 |
| 監査 | 脆弱性、秘密情報、危険なパターンをスキャンします | 稼働中 | 🔄 |
`

const longTokenTable = `| Endpoint | Redirect target | Notes |
|----------|-----------------|-------|
| /v1/completions | https://api.example.com/v1/chat/completions?stream=true&model=glm-5.3-flash | passthrough |
| /v1/long | supercalifragilisticexpialidociousness_and_then_some_more_letters_to_force_a_hard_break | no spaces at all |
| /v1/empty |  | deliberately empty cell |
`

const manyRowsTable = `| # | Span | Dur ms | In | Out | Cached | Err |
|--:|------|-------:|----:|----:|-------:|-----|
| 1 | chat.completion | 4120 | 1204 | 318 | 0 |  |
| 2 | tool.execute | 87 | 41 | 12 | 0 |  |
| 3 | chat.completion | 9610 | 2380 | 744 | 1024 |  |
| 4 | tool.execute | 4 | 9 | 3 | 0 |  |
| 5 | chat.completion | 15233 | 4011 | 1205 | 2048 |  |
| 6 | chat.completion | 2201 | 880 | 210 | 512 | timeout |
| 7 | tool.execute | 31 | 22 | 8 | 0 |  |
| 8 | chat.completion | 7788 | 1943 | 566 | 1024 |  |
| 9 | tool.execute | 512 | 66 | 19 | 0 |  |
| 10 | chat.completion | 30440 | 6120 | 2090 | 4096 |  |
| 11 | subagent.dispatch | 61204 | 340 | 88 | 0 |  |
| 12 | chat.completion | 1190 | 402 | 140 | 0 |  |
`

const document = `## Dispatch results

Each row below is one sub-agent dispatch from the last run. The table is
narrow on purpose: five columns, and the summary column carries real prose
so the layout has something to compress.

| Role | Outcome | Summary | Turns |
|------|---------|---------|------:|
| developer | proposal | Fixed the icon collision and pinned exact glyphs in the known-icons test | 6 |
| reviewer | clean | No findings; confirmed the two prior issues were already closed by the branch under review | 9 |
| tester | pass | 291 of 291 tests green, race detector clean, coverage unchanged at 87.4 percent | 3 |

Follow-up work is tracked in the issue database, not here.
`
