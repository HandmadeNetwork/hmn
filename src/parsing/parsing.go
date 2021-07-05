package parsing

import (
	"bytes"

	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/util"
)

// Used for rendering real-time previews of post content.
var PreviewMarkdown = goldmark.New(
	goldmark.WithExtensions(makeGoldmarkExtensions(true)...),
)

// Used for generating the final HTML for a post.
var RealMarkdown = goldmark.New(
	goldmark.WithExtensions(makeGoldmarkExtensions(false)...),
)

// Used for generating plain-text previews of posts.
var PlaintextMarkdown = goldmark.New(
	goldmark.WithExtensions(makeGoldmarkExtensions(false)...),
	goldmark.WithRenderer(plaintextRenderer{}),
)

func ParsePostInput(source string, md goldmark.Markdown) string {
	var buf bytes.Buffer
	if err := md.Convert([]byte(source), &buf); err != nil {
		panic(err)
	}

	return buf.String()
}

func makeGoldmarkExtensions(preview bool) []goldmark.Extender {
	return []goldmark.Extender{
		extension.GFM,
		highlightExtension,
		SpoilerExtension{},
		EmbedExtension{
			Preview: preview,
		},
		MathjaxExtension{},
		BBCodeExtension{
			Preview: preview,
		},
	}
}

var highlightExtension = highlighting.NewHighlighting(
	highlighting.WithFormatOptions(HMNChromaOptions...),
	highlighting.WithWrapperRenderer(func(w util.BufWriter, context highlighting.CodeBlockContext, entering bool) {
		if entering {
			w.WriteString(`<pre class="hmn-code">`)
		} else {
			w.WriteString(`</pre>`)
		}
	}),
)
