package parsing

import (
	"bytes"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

var previewMarkdown = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,
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
