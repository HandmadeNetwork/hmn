package parsing

import (
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
