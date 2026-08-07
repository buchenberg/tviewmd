# ADR 0001: Streaming Markdown Rendering for LLM Responses

## Status

**Accepted** 

## Context

tviewmd was originally designed for batch processing of complete markdown documents. Users want to render LLM (Large Language Model) responses in real-time as they stream in, rather than waiting for the complete response.

The current architecture has these characteristics:
- `Parse()` expects a complete markdown string
- Goldmark parser works on complete documents
- `Render()` and `RenderTView()` assume complete blocks
- No state management between parsing calls

## Problem

When rendering streaming LLM output:
1. **Partial blocks break parsing**: Incomplete markdown (e.g., open code fence without close, unfinished list) parses incorrectly
2. **No incremental API**: Cannot efficiently add new content to existing rendered output
3. **Performance concerns**: Full re-parsing of growing buffer on each chunk is O(n²)

## Decision

**Adopt a buffered re-render approach for v1.x, with incremental parser as future enhancement.**

### Current Recommendation (v1.x)

Use **buffer + throttled re-render** pattern:

```go
func renderLLMStream(stream <-chan []byte, tv *tview.TextView) {
    var buf strings.Builder
    throttle := time.Tick(50 * time.Millisecond)

    for {
        select {
        case chunk := <-stream:
            buf.Write(chunk)
        case <-throttle:
            tv.SetText(tviewmd.Render(buf.String(), opts))
        }
    }
}
```

**Rationale**:
- ✅ Zero library changes required
- ✅ Always produces correct output
- ✅ Fuzz testing confirms panic-free on arbitrary input
- ✅ Simple to implement for users
- ⚠️ O(n²) performance for large responses (mitigated by throttling)

### Future Enhancement (v2.0+)

Add **incremental parsing** support:

```go
// Proposed API
type StreamRenderer struct {
    blocks   []Block
    pending  string
    opts     Options
}

func (s *StreamRenderer) Write(chunk []byte) string
func (s *StreamRenderer) Flush() string
```

**Rationale for deferral**:
- Significant implementation complexity (~200-300 lines)
- Goldmark doesn't natively support incremental parsing
- Need to handle: partial code blocks, incomplete lists, unterminated tables
- Current buffer approach is "good enough" for most use cases

## Consequences

### Positive
- **Immediate usability**: Users can stream LLM responses today with existing API
- **Correctness**: Buffer approach guarantees correct rendering
- **Simplicity**: No new public API surface in v1.x
- **Safety**: Fuzz-tested parser handles arbitrary partial input without crashing

### Negative
- **Performance**: Throttled re-render is O(n²) but acceptable for typical LLM response sizes
- **No partial block handling**: Incomplete markdown at chunk boundaries renders as plain text until complete
- **Memory**: Buffer grows with response size (mitigated by streaming nature)

## Alternatives Considered

### Alternative 1: Chunked with Heuristic Boundary Detection
Attempt to detect when a markdown block is complete and render it immediately.

**Rejected because**:
- Complex heuristics for detecting block completeness
- Still breaks on edge cases (nested lists, multi-line code blocks)
- Error-prone and hard to maintain

### Alternative 2: Incremental Parser in v1.x
Implement full incremental parsing support immediately.

**Rejected because**:
- Delays v1.0.0 release
- Significant implementation effort for limited immediate benefit
- Most users can work with buffer approach
- Adds complexity to core library

### Alternative 3: Separate Streaming Package
Create `tviewmd/stream` subpackage with streaming support.

**Rejected because**:
- Adds complexity to module structure
- Streaming is a core use case, not a separate concern
- Better to enhance main package

## Related

- Issue: Streaming support for LLM responses
- Fuzz testing in `fuzz_test.go` validates panic-free parsing of arbitrary input
- Goldmark parser is safe for concurrent use (shared instance in `parse.go`)

## Notes

The buffer + throttled re-render approach has been validated with:
- Chat applications rendering assistant responses
- Code generation tools showing streaming output
- Real-time documentation generation

Future incremental parser implementation should:
1. Parse complete blocks from input stream
2. Buffer partial blocks until completion
3. Handle timeout-based flushing for stuck streams
4. Maintain backward compatibility with batch API

---

*Created: 2026-08-07*  
*Last Updated: 2026-08-07*  
*Status: Accepted*