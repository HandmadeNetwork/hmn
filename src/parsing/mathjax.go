package parsing

import (
	gohtml "html"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
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

type mathjaxParser struct{}

var _ parser.BlockParser = mathjaxParser{}

func (s mathjaxParser) Trigger() []byte {
	return []byte{'$'}
}

func (s mathjaxParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, _ := reader.PeekLine()
	lineStr := strings.TrimSpace(string(line))

	if lineStr == "$$" {
		return NewMathjax(), parser.NoChildren
	} else {
		return nil, parser.NoChildren
	}
}

func (s mathjaxParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	line, _ := reader.PeekLine()
	lineStr := strings.TrimSpace(string(line))

	if lineStr == "$$" {
		reader.Advance(len(line))
		return parser.Close
	}

	node.(*MathjaxNode).Source += string(line)
	return parser.Continue | parser.NoChildren
}

func (s mathjaxParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {}

func (s mathjaxParser) CanInterruptParagraph() bool {
	return true
}

func (s mathjaxParser) CanAcceptIndentedLine() bool {
	return false
}

// ----------------------
// AST node
// ----------------------

type MathjaxNode struct {
	gast.BaseBlock
	Source string
}

var _ gast.Node = &MathjaxNode{}

func (n *MathjaxNode) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, nil, nil)
}

var KindMathjax = gast.NewNodeKind("Mathjax")

func (n *MathjaxNode) Kind() gast.NodeKind {
	return KindMathjax
}

func NewMathjax() *MathjaxNode {
	return &MathjaxNode{}
}

// ----------------------
// Renderer
// ----------------------

type MathjaxHTMLRenderer struct {
	html.Config
}

func NewMathjaxHTMLRenderer(opts ...html.Option) renderer.NodeRenderer {
	r := &MathjaxHTMLRenderer{
		Config: html.NewConfig(),
	}
	for _, opt := range opts {
		opt.SetHTMLOption(&r.Config)
	}
	return r
}

func (r *MathjaxHTMLRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindMathjax, r.renderMathjax)
}

func (r *MathjaxHTMLRenderer) renderMathjax(w util.BufWriter, source []byte, n gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		w.WriteString("<div>\n")
		w.WriteString("$$\n")
		w.WriteString(gohtml.EscapeString(n.(*MathjaxNode).Source))
		w.WriteString("$$\n")
		w.WriteString("</div>\n")
	}
	return gast.WalkSkipChildren, nil
}

// ----------------------
// Extension
// ----------------------

type MathjaxExtension struct {
	Preview bool
}

func (e MathjaxExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(parser.WithBlockParsers(
		util.Prioritized(mathjaxParser{}, 500),
	))
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(NewMathjaxHTMLRenderer(), 500),
	))
}
