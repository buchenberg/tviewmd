package tviewmd

import (
	"strings"
	"testing"
)

var benchMarkdown = strings.Repeat(`# Heading

A paragraph with **bold**, *italic*, `+"`code`"+`, and [a link](https://example.com).

| Feature | Status |
|---------|--------|
| retained proposals | shipped |
| janitor scan/run | shipped |

> - quoted list
> - second item

`, 20)

func BenchmarkRender(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = Render(benchMarkdown, Options{})
	}
}

func BenchmarkRenderTViewPreparsed(b *testing.B) {
	blocks, err := Parse(benchMarkdown)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = RenderTView(blocks, Options{})
	}
}

func BenchmarkRendererAppendString(b *testing.B) {
	chunks := strings.Split(benchMarkdown, " ")
	r := NewRenderer(Options{})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Append(chunks[i%len(chunks)])
		_ = r.String()
	}
}
