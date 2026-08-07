// Package tviewmd renders markdown to tview color-tagged text.
//
// It parses CommonMark + GFM markdown (via goldmark) into a renderer-neutral
// block/segment model, then emits tview color tags directly — suitable for a
// tview.TextView with SetWrap(true). No ANSI round-trip is required.
//
// The primary entry points are [Render] (parse + render) and [RenderTView]
// (render pre-parsed blocks). Callers that want to inspect or transform the
// document can use [Parse] to obtain the intermediate []Block representation.
package tviewmd
