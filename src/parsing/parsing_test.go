package parsing

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseBBCode(t *testing.T) {
	const testDoc = `Hello, [b]amazing[/b] [i]incredible[/i] [b][i][u]world!!![/u][/i][/b]

Too many opening tags: [b]wow.[b]

Too many closing tags: [/i]wow.[/i]

Mix 'em: [u][/i]wow.[/i][/u]

[url=https://google.com/]Google![/url]
`

	t.Run("hello world", func(t *testing.T) {
		bbcode := "Hello, [b]amazing[/b] [i]incredible[/i] [b][i][u]world!!![/u][/i][/b]"
		expected := "Hello, <strong>amazing</strong> <em>incredible</em> <strong><em><u>world!!!</u></em></strong>"
		assert.Equal(t, expected, ParseBBCode(bbcode))
	})
}

// func TestParsePostInput(t *testing.T) {
// 	testDoc := []byte(`
// Hello, *world!*

// I can do **bold**, *italic*, and _underlined_ text??

// # Heading 1
// ## Heading 2
// ### Heading 3

// Links: [HMN](https://handmade.network)
// Images: ![this is a picture of sanic](https://i.kym-cdn.com/photos/images/newsfeed/000/722/711/ef1.jpg)
// `)

// 	res := ParsePostInput(testDoc)
// 	fmt.Println(string(res))

// 	t.Fail()
// }

func TestBBCodeParsing(t *testing.T) {
	res := ParsePostInput(`[b]ONE[/b] [i]TWO[/i]`, false)
	fmt.Println(res)
	t.Fail()
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
`
