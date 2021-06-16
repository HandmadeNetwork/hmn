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
	res := ParsePostInput(`[b]ONE[/b] [i]TWO[/i]`)
	fmt.Println(res)
	t.Fail()
}
