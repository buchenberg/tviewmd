# ADR 0001: Streaming Markdown Rendering for LLM Responses

## Status

**Accepted** (proposed 2026-08-07; updated 2026-11-22 to reflect the shipped `Renderer`)

## Context

tviewmd was originally designed for batch processing of complete markdown documents. Users want to render LLM (Large Language Model) responses in real-time as they stream in, rather than waiting for the complete response.

The architecture has these characteristics:

- `Parse()` expects a complete markdown string.
- Goldmark parses complete documents; it has no incremental mode.
- `Render()` and `RenderTView()` render whole blocks.
- There is no state management between parsing calls (historically).

## Problem

When rendering streaming LLM output:

1. **Partial blocks break parsing**: incomplete markdown (e.g., an open code fence without a close, an unfinished list) parses differently from the final document.
2. **No incremental API**: originally there was no way to add new content to an existing rendered document without re-feeding it.
3. **Performance concerns**: full re-parsing of a growing buffer on every chunk is O(n²).

## Decision

**Use a buffered, throttled re-render.** Re-render the entire accumulated source on a tick from the tview event loop. This always produces the same output as a final full render and avoids the raw→formatted flash that line- or block-boundary splitting causes.

This pattern is now a first-class part of the library: `Renderer` accumulates source with `Append` / `Write` (it implements `io.Writer`), caches `String()`, and supports `SetWidth` / `SetOptions` for resize and theme changes without re-feeding the stream. `Err` reports `ErrStreamTooLarge` when `Options.MaxStreamBytes` (default 8 MiB, `-1` for unlimited) caps a runaway stream. `Renderer` is safe for concurrent use, so a producer goroutine can append while the event loop flushes.

Reference pattern (channel-close handling included):

```go
func renderLLMStream(app *tview.Application, stream <-chan []byte, tv *tview.TextView) {
    // tview disables dynamic colors by default; without this the emitted tags
    // render as literal text.
    tv.SetWrap(true).SetDynamicColors(true)

    r := tviewmd.NewRenderer(tviewmd.Options{})
    ticker := time.NewTicker(50 * time.Millisecond)
    defer ticker.Stop()

    for {
        select {
        case chunk, ok := <-stream:
            if !ok {
                // Closed stream: flush the final buffer and exit. Reading from
                // a closed channel yields nil chunks forever otherwise.
                app.QueueUpdateDraw(func() { tv.SetText(r.String()) })
                return
            }
            r.Append(string(chunk))
        case <-ticker.C:
            app.QueueUpdateDraw(func() { tv.SetText(r.String()) })
        }
    }
}
```

Do not call `QueueUpdateDraw` (or the blocking `QueueUpdate`) from inside an event-loop callback: the loop cannot run the queued function while executing the current one. When already on the event-loop goroutine, assign to the `TextView` directly.

### Resolved proposal

An earlier draft proposed a separate v2.0 `StreamRenderer` with `Write(chunk []byte) string` / `Flush() string` and incremental parsing. That proposal is **superseded by `Renderer`**: incremental parsing was not required for correct output, and the buffered path now lives in the library with a stream cap, cached output and reconfiguration hooks. No separate incremental-parser API is planned.

## Consequences

### Positive

- **Immediate usability**: callers can stream LLM responses today using `Renderer` or their own buffer.
- **Correctness**: re-rendering the whole buffer always matches a final full render.
- **Reusable API surface**: `Renderer`, `ErrStreamTooLarge` and `Options.MaxStreamBytes` ship in the main package.
- **Safety**: the fuzz-tested parser handles arbitrary partial input without crashing.

### Negative

- **Performance**: throttled re-render is O(n²) in the worst case, mitigated by throttling and `String()` caching.
- **No partial-block rendering**: an incomplete code fence or list renders as plain text until its closer arrives.
- **Memory**: the buffer grows with response size unless bounded by `Options.MaxStreamBytes`.

## Alternatives Considered

### Alternative 1: Chunked with heuristic boundary detection

Detect when a markdown block is complete and render it immediately.

**Rejected because**: complex heuristics for block completeness, still breaks on nested lists and multi-line code fences, and is hard to maintain.

### Alternative 2: Incremental parser

Implement full incremental parsing in the core library.

**Rejected because**: goldmark has no incremental mode, partial code blocks / lists / tables need extensive special-casing, and it adds significant complexity to the core library for limited benefit. `Renderer` covers the practical use case.

### Alternative 3: Separate streaming package

Create a `tviewmd/stream` subpackage with streaming support.

**Rejected because**: streaming is a core use case, not a separate concern, and a subpackage adds module complexity without benefit over the main-package `Renderer`.

## Related

- Issue: Streaming support for LLM responses
- README: "Streaming LLM Output" for the `Renderer` walkthrough
- `fuzz_test.go` validates panic-free parsing of arbitrary input
- Goldmark's parser is used concurrently through a shared instance (`parse.go`)

## Notes

The buffer + throttled re-render approach has been validated with chat applications rendering assistant responses, code-generation tools showing streaming output, and real-time documentation generation. The incremental-parser follow-up listed in the original draft was resolved by shipping `Renderer` (see "Resolved proposal" above).

---

*Created: 2026-08-07*  
*Last Updated: 2026-11-22*  
*Status: Accepted*
