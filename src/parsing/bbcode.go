package parsing

import (
	"fmt"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

const BBCodePriority = 1 // TODO: Pick something more reasonable?

type bParser struct{}

var _ parser.InlineParser = bParser{}

func (s bParser) Trigger() []byte {
	return []byte{'['}
}

func (s bParser) Parse(parent gast.Node, block text.Reader, pc parser.Context) gast.Node {
	// _, segment := block.PeekLine()
	// start := segment.Start

	// block.Advance(3)
	// n := ast.NewTextSegment(text.NewSegment(start, start+4))
	// bold := ast.NewText()
	// bold.Segment
	// link := ast.NewAutoLink(typ, n)
	// link.Protocol = protocol
	// return link

	lineBytes, segment := block.PeekLine()
	fmt.Printf("line: %s\n", string(lineBytes))
	fmt.Printf("segment: %#v\n", segment)

	line := string(lineBytes)

	if !strings.HasPrefix(line, "[b]") {
		return nil
	}
	start := 0

	closingIndex := strings.Index(line, "[/b]")
	if closingIndex < 0 {
		return nil
	}
	end := closingIndex + 4

	n := ast.NewEmphasis(2)
	n.AppendChild(n, ast.NewString([]byte("wow bold text")))

	block.Advance(end - start)
	return n
}

type bTag struct{}

func (e bTag) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(parser.WithInlineParsers(
		util.Prioritized(bParser{}, BBCodePriority),
	))
	// m.Renderer().AddOptions(renderer.WithNodeRenderers(
	// 	util.Prioritized(NewStrikethroughHTMLRenderer(), 500),
	// ))
}
