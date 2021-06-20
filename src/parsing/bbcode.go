package parsing

import (
	"regexp"

	"github.com/frustra/bbcode"
	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

var BBCodePriority = 1 // TODO: This is maybe too high a priority?

var reTag = regexp.MustCompile(`(?P<open>\[\s*(?P<opentagname>[a-zA-Z]+))|(?P<close>\[\s*\/\s*(?P<closetagname>[a-zA-Z]+)\s*\])`)

var bbcodeCompiler = bbcode.NewCompiler(false, false)

// ----------------------
// Parser and delimiters
// ----------------------

type bbcodeParser struct{}

func NewBBCodeParser() parser.InlineParser {
	return bbcodeParser{}
}

func (s bbcodeParser) Trigger() []byte {
	return []byte{'['}
}

func (s bbcodeParser) Parse(parent gast.Node, block text.Reader, pc parser.Context) gast.Node {
	_, pos := block.Position()
	restOfSource := block.Source()[pos.Start:]

	matches := reTag.FindAllSubmatchIndex(restOfSource, -1)
	if matches == nil {
		// No tags anywhere
		return nil
	}

	otIndex := reTag.SubexpIndex("opentagname")
	ctIndex := reTag.SubexpIndex("closetagname")

	tagName := extractStringBySubmatchIndices(restOfSource, matches[0], otIndex)
	if tagName == "" {
		// Not an opening tag
		return nil
	}

	depth := 0
	endIndex := -1
	for _, m := range matches {
		if openName := extractStringBySubmatchIndices(restOfSource, m, otIndex); openName != "" {
			if openName == tagName {
				depth++
			}
		} else if closeName := extractStringBySubmatchIndices(restOfSource, m, ctIndex); closeName != "" {
			if closeName == tagName {
				depth--
				if depth == 0 {
					// We have balanced out!
					endIndex = m[1] // the end index of this closing tag (exclusive)
				}
			}
		}
	}
	if endIndex < 0 {
		// Unbalanced, too many opening tags
		return nil
	}

	unparsedBBCode := restOfSource[:endIndex]
	block.Advance(len(unparsedBBCode))

	return NewBBCode(bbcodeCompiler.Compile(string(unparsedBBCode)))
}

func extractStringBySubmatchIndices(src []byte, m []int, subexpIndex int) string {
	srcIndices := m[2*subexpIndex : 2*subexpIndex+1+1]
	if srcIndices[0] < 0 {
		return ""
	}
	return string(src[srcIndices[0]:srcIndices[1]])
}

// ----------------------
// AST node
// ----------------------

type BBCodeNode struct {
	gast.BaseInline
	HTML string
}

var _ gast.Node = &BBCodeNode{}

func (n *BBCodeNode) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, nil, nil)
}

var KindBBCode = gast.NewNodeKind("BBCode")

func (n *BBCodeNode) Kind() gast.NodeKind {
	return KindBBCode
}

func NewBBCode(html string) gast.Node {
	return &BBCodeNode{
		HTML: html,
	}
}

// ----------------------
// Renderer
// ----------------------

type BBCodeHTMLRenderer struct {
	html.Config
}

func NewBBCodeHTMLRenderer(opts ...html.Option) renderer.NodeRenderer {
	r := &BBCodeHTMLRenderer{
		Config: html.NewConfig(),
	}
	for _, opt := range opts {
		opt.SetHTMLOption(&r.Config)
	}
	return r
}

func (r *BBCodeHTMLRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindBBCode, r.renderBBCode)
}

func (r *BBCodeHTMLRenderer) renderBBCode(w util.BufWriter, source []byte, n gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		w.WriteString(n.(*BBCodeNode).HTML)
	}
	return gast.WalkContinue, nil
}

// ----------------------
// Extension
// ----------------------

type BBCodeExtension struct{}

func (e BBCodeExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(parser.WithInlineParsers(
		util.Prioritized(NewBBCodeParser(), BBCodePriority),
	))
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(NewBBCodeHTMLRenderer(), BBCodePriority),
	))
}
