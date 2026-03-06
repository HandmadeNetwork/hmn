package parsing

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMarkdown(t *testing.T) {
	t.Run("fenced code blocks", func(t *testing.T) {
		t.Run("multiple lines", func(t *testing.T) {
			html := ParseMarkdown("```\nmultiple lines\n\tof code\n```", PostMarkdown)
			t.Log(html)
			assert.Equal(t, 1, strings.Count(html, "<pre"))
			assert.Contains(t, html, `class="hmn-code"`)
			assert.Contains(t, html, "multiple lines\n\tof code")
		})
		t.Run("multiple lines with language", func(t *testing.T) {
			html := ParseMarkdown("```go\nfunc main() {\n\tfmt.Println(\"Hello, world!\")\n}\n```", PostMarkdown)
			t.Log(html)
			assert.Equal(t, 1, strings.Count(html, "<pre"))
			assert.Contains(t, html, `class="hmn-code"`)
			assert.Contains(t, html, "Println")
			assert.Contains(t, html, "Hello, world!")
		})
	})
}

func TestMarkdownVariants(t *testing.T) {
	md := `
Here's a cool image that I really like a lot.

![picture of a dog](coolimage.png)

And here's my favorite YouTube video:

https://youtu.be/dQw4w9WgXcQ

I hope you like it as much as I do.
`
	t.Run("Real post Markdown", func(t *testing.T) {
		html := ParseMarkdown(md, PostMarkdown)
		t.Log(html)
		assert.Contains(t, html, `<img src="coolimage.png"`)
		assert.Contains(t, html, "<iframe")
	})
	t.Run("Post edit preview Markdown", func(t *testing.T) {
		html := ParseMarkdown(md, PostEditPreviewMarkdown)
		t.Log(html)
		assert.Contains(t, html, `<img src="coolimage.png"`)
		assert.Contains(t, html, `<img src="https://img.youtube.com/vi/dQw4w9WgXcQ/hqdefault.jpg"`)
		assert.NotContains(t, html, "<iframe")
	})
	t.Run("Post preview Markdown", func(t *testing.T) {
		html := ParseMarkdown(md, PostPreviewMarkdown)
		t.Log(html)
		assert.Contains(t, html, `<img src="coolimage.png"`)
		assert.Contains(t, html, `<a href="https://youtu.be/dQw4w9WgXcQ"`)
		assert.NotContains(t, html, "<iframe")
	})
	t.Run("Plaintext Markdown", func(t *testing.T) {
		html := ParseMarkdown(md, PlaintextMarkdown)
		t.Log(html)
		assert.NotContains(t, html, `<img`)
		assert.NotContains(t, html, `<a`)
		assert.NotContains(t, html, "<iframe")
		assert.NotContains(t, html, "\n", "Plain text markdown is intended for OpenGraph descriptions and therefore shouldn't contain newlines")
	})
}

func TestBBCode(t *testing.T) {
	t.Run("[code]", func(t *testing.T) {
		t.Run("one line", func(t *testing.T) {
			html := ParseMarkdown("[code]Just some code, you know?[/code]", PostMarkdown)
			t.Log(html)
			assert.Equal(t, 1, strings.Count(html, "<pre"))
			assert.Contains(t, html, `class="hmn-code"`)
			assert.Contains(t, html, "Just some code, you know?")
		})
		t.Run("multiline", func(t *testing.T) {
			bbcode := `[code]
Multiline code
	with an indent
[/code]`
			html := ParseMarkdown(bbcode, PostMarkdown)
			t.Log(html)
			assert.Equal(t, 1, strings.Count(html, "<pre"))
			assert.Contains(t, html, `class="hmn-code"`)
			assert.Contains(t, html, "Multiline code\n\twith an indent")
			assert.NotContains(t, html, "<br")
		})
		t.Run("multiline with language", func(t *testing.T) {
			bbcode := `[code language=go]
func main() {
	fmt.Println("Hello, world!")
}
[/code]`
			html := ParseMarkdown(bbcode, PostMarkdown)
			t.Log(html)
			assert.Equal(t, 1, strings.Count(html, "<pre"))
			assert.Contains(t, html, "Println")
			assert.Contains(t, html, "Hello, world!")
		})
	})
}

func TestSharlock(t *testing.T) {
	t.Skipf("This doesn't pass right now because parts of Sharlock's original source read as indented code blocks, or depend on different line break behavior.")
	t.Run("sanity check", func(t *testing.T) {
		result := ParseMarkdown(sharlock, PostMarkdown)

		for line := range strings.SplitSeq(result, "\n") {
			assert.NotContains(t, line, "[b]")
			assert.NotContains(t, line, "[/b]")
			assert.NotContains(t, line, "[ul]")
			assert.NotContains(t, line, "[/ul]")
			assert.NotContains(t, line, "[li]")
			assert.NotContains(t, line, "[/li]")
			assert.NotContains(t, line, "[img]")
			assert.NotContains(t, line, "[/img]")
			assert.NotContains(t, line, "[code")
			assert.NotContains(t, line, "[/code]")
		}
	})
}

func BenchmarkSharlock(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ParseMarkdown(sharlock, PostMarkdown)
	}
}
