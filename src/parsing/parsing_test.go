package parsing

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMarkdown(t *testing.T) {
	t.Run("fenced code blocks", func(t *testing.T) {
		t.Run("multiple lines", func(t *testing.T) {
			html := ParsePostInput("```\nmultiple lines\n\tof code\n```", RealMarkdown)
			t.Log(html)
			assert.Equal(t, 1, strings.Count(html, "<pre"))
			assert.Contains(t, html, `class="hmn-code"`)
			assert.Contains(t, html, "multiple lines\n\tof code")
		})
		t.Run("multiple lines with language", func(t *testing.T) {
			html := ParsePostInput("```go\nfunc main() {\n\tfmt.Println(\"Hello, world!\")\n}\n```", RealMarkdown)
			t.Log(html)
			assert.Equal(t, 1, strings.Count(html, "<pre"))
			assert.Contains(t, html, `class="hmn-code"`)
			assert.Contains(t, html, "Println")
			assert.Contains(t, html, "Hello, world!")
		})
	})
}

func TestBBCode(t *testing.T) {
	t.Run("[code]", func(t *testing.T) {
		t.Run("one line", func(t *testing.T) {
			html := ParsePostInput("[code]Just some code, you know?[/code]", RealMarkdown)
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
			html := ParsePostInput(bbcode, RealMarkdown)
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
			html := ParsePostInput(bbcode, RealMarkdown)
			t.Log(html)
			assert.Equal(t, 1, strings.Count(html, "<pre"))
			assert.Contains(t, html, "Println")
			assert.Contains(t, html, "Hello, world!")
		})
	})
}

const allBBCode = `
[b]bold[/b]

[i]italic[/i]

[u]underline[/u]

[h1]heading 1[/h1]

[h2]heading 2[/h2]

[h3]heading 3[/h3]

[m]monospace[/m]

[ol]
  [li]ordered lists[/li]
[/ol]

[ul]
  [li]unordered list[/li]
[/ul]

[url]https://handmade.network/[/url]
[url=https://handmade.network/]Handmade Network[/url]

[img=https://handmade.network/static/media/members/avatars/delix.jpeg]Ryan[/img]

[quote]quotes[/quote]
[quote=delix]Some quote[/quote]

[code]
Code
[/code]

[code language=go]
func main() {
  fmt.Println("Hello, world!")
}
[/code]

[spoiler]spoilers[/spoiler]

[table]
[tr]
[th]Heading 1[/th] [th]Heading 2[/th]
[/tr]
[tr]
[td]Body 1[/td] [td]Body 2[/td]
[/tr]
[/table]

[youtube]https://www.youtube.com/watch?v=0J8G9qNT7gQ[/youtube]
[youtube]https://youtu.be/0J8G9qNT7gQ[/youtube]
`
