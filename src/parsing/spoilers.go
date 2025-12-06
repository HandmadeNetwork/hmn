package parsing

import (
	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// ----------------------
// Parser and delimiters
// ----------------------

type spoilerParser struct{}

func NewSpoilerParser() parser.InlineParser {
	return spoilerParser{}
}

func (s spoilerParser) Trigger() []byte {
	return []byte{'|'}
}

func (s spoilerParser) Parse(parent gast.Node, block text.Reader, pc parser.Context) gast.Node {
	before := block.PrecendingCharacter()                                              // ScanDelimiter needs this for shady left vs. right delimiter reasons. Who delimits the delimiters?
	restOfLine, segment := block.PeekLine()                                            // Gets the rest of the line (starting at the current parser cursor index), and the segment representing the indices in the source text.
	delimiter := parser.ScanDelimiter(restOfLine, before, 2, spoilerDelimiterParser{}) // Scans a consecutive run of the trigger character. Returns a delimiter node tracking which character that was, how many of it there were, etc. We do 2 here because we want ~~spoilers~~.
	if delimiter == nil {
		// I guess we only saw one ~ :)
		return nil
	}
	delimiter.Segment = segment.WithStop(segment.Start + delimiter.OriginalLength) // The delimiter needs to know exactly what source indices it corresponds to.
	block.Advance(delimiter.OriginalLength)                                        // Advance the parser past the delimiter.
	pc.PushDelimiter(delimiter)                                                    // Push the delimiter onto the stack (either opening or closing; both are handled the same way as far as this method is concerned).
	return delimiter
}

type spoilerDelimiterParser struct{}

func (p spoilerDelimiterParser) IsDelimiter(b byte) bool {
	return b == '|'
}

func (p spoilerDelimiterParser) CanOpenCloser(opener, closer *parser.Delimiter) bool {
	return opener.Char == closer.Char
}

func (p spoilerDelimiterParser) OnMatch(consumes int) gast.Node {
	return NewSpoiler()
}

// ----------------------
// AST node
// ----------------------

type SpoilerNode struct {
	gast.BaseInline
}

var _ gast.Node = &SpoilerNode{}

func (n *SpoilerNode) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, nil, nil)
}

var KindSpoiler = gast.NewNodeKind("Spoiler")

func (n *SpoilerNode) Kind() gast.NodeKind {
	return KindSpoiler
}

func NewSpoiler() gast.Node {
	return &SpoilerNode{}
}

// ----------------------
// Renderer
// ----------------------

type SpoilerHTMLRenderer struct {
	html.Config
}

func NewSpoilerHTMLRenderer(opts ...html.Option) renderer.NodeRenderer {
	r := &SpoilerHTMLRenderer{
		Config: html.NewConfig(),
	}
	for _, opt := range opts {
		opt.SetHTMLOption(&r.Config)
	}
	return r
}

func (r *SpoilerHTMLRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindSpoiler, r.renderSpoiler)
}

func (r *SpoilerHTMLRenderer) renderSpoiler(w util.BufWriter, source []byte, n gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		_, _ = w.WriteString("<span class=\"spoiler\">")
	} else {
		_, _ = w.WriteString("</span>")
	}
	return gast.WalkContinue, nil
}

// ----------------------
// Extension
// ----------------------

type SpoilerExtension struct{}

func (e SpoilerExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(parser.WithInlineParsers(
		util.Prioritized(NewSpoilerParser(), 500),
	))
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(NewSpoilerHTMLRenderer(), 500),
	))
}
