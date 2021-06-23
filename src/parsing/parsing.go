package parsing

import (
	"bytes"

	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/util"
)

var previewMarkdown = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,
		highlightExtension,
		SpoilerExtension{},
		EmbedExtension{
			Preview: true,
		},
		MathjaxExtension{},
		BBCodeExtension{
			Preview: true,
		},
	),
)
var realMarkdown = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,
		highlightExtension,
		SpoilerExtension{},
		EmbedExtension{},
		MathjaxExtension{},
		BBCodeExtension{},
	),
)

func ParsePostInput(source string, preview bool) string {
	md := realMarkdown
	if preview {
		md = previewMarkdown
	}

	var buf bytes.Buffer
	if err := md.Convert([]byte(source), &buf); err != nil {
		panic(err)
	}

	return buf.String()
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
