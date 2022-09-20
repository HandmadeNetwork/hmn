package parsing

import (
	"bytes"
	"fmt"
	"regexp"

	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/utils"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// NOTE(ben): ggcode is my cute name for our custom extension syntax because I got fed up with
// bbcode. It's designed to be a more natural fit for Goldmark's method of parsing, while still
// being a general-purpose tag-like syntax that's easy for us to add instances of without writing
// new Goldmark parsers.
//
// Inline ggcode is delimited by two exclamation marks. Block ggcode is delimited by three. Inline
// ggcode uses parentheses to delimit the start and end of the affected content. Block ggcode is
// like a fenced code block and ends with !!!. ggcode sections can optionally have named string
// arguments inside braces. Quotes around the value are mandatory.
//
// Inline example:
//
//     See our article on !!glossary{slug="tcp"}(TCP) for more details.
//
// Block example:
//
//     !!!resource{name="Beej's Guide to Network Programming" url="https://beej.us/guide/bgnet/html/"}
//     This is a _fantastic_ resource on network programming, suitable for beginners.
//     !!!
//

var ggcodeTags = map[string]ggcodeTag{
	"glossary": {
		Filter: ggcodeFilterEdu,
		Renderer: func(c ggcodeRendererContext, n *ggcodeNode, entering bool) error {
			if entering {
				term, _ := n.Args["term"]
				c.W.WriteString(fmt.Sprintf(
					`<a href="%s" class="glossary-term" data-term="%s">`,
					hmnurl.BuildEducationGlossary(term),
					term,
				))
			} else {
				c.W.WriteString("</a>")
			}
			return nil
		},
	},
	"resource": {
		Filter: ggcodeFilterEdu,
		Renderer: func(c ggcodeRendererContext, n *ggcodeNode, entering bool) error {
			if entering {
				c.W.WriteString(`<div class="edu-resource">`)
				c.W.WriteString(fmt.Sprintf(`  <a class="resource-title" href="%s" target="_blank">%s</a>`, n.Args["url"], utils.OrDefault(n.Args["name"], "[missing `name`]")))
			} else {
				c.W.WriteString("</div>")
			}
			return nil
		},
	},
	"note": {
		Filter: ggcodeFilterEdu,
		Renderer: func(c ggcodeRendererContext, n *ggcodeNode, entering bool) error {
			if entering {
				c.W.WriteString(`<span class="note">`)
			} else {
				c.W.WriteString(`</span>`)
			}
			return nil
		},
	},
}

// ----------------------
// Types
// ----------------------

type ggcodeRendererContext struct {
	W      util.BufWriter
	Source []byte
	Opts   MarkdownOptions
}

type ggcodeTagFilter func(opts MarkdownOptions) bool
type ggcodeRenderer func(c ggcodeRendererContext, n *ggcodeNode, entering bool) error

type ggcodeTag struct {
	Filter   ggcodeTagFilter
	Renderer ggcodeRenderer
}

var ggcodeFilterEdu ggcodeTagFilter = func(opts MarkdownOptions) bool {
	return opts.Education
}

// ----------------------
// Parsers and delimiters
// ----------------------

var reGGCodeBlockOpen = regexp.MustCompile(`^!!!(?P<name>[a-zA-Z0-9-_]+)(\{(?P<args>.*?)\})?$`)
var reGGCodeInline = regexp.MustCompile(`^!!(?P<name>[a-zA-Z0-9-_]+)(\{(?P<args>.*?)\})?(\((?P<content>.*?)\))?`)
var reGGCodeArgs = regexp.MustCompile(`(?P<arg>[a-zA-Z0-9-_]+)="(?P<val>.*?)"`)

// Block parser stuff

type ggcodeBlockParser struct{}

var _ parser.BlockParser = ggcodeBlockParser{}

func (s ggcodeBlockParser) Trigger() []byte {
	return []byte("!")
}

func (s ggcodeBlockParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	restOfLine, _ := reader.PeekLine()

	if match := extractMap(reGGCodeBlockOpen, bytes.TrimSpace(restOfLine)); match != nil {
		name := string(match["name"])
		var args map[string]string
		if argsMatch := extractAllMap(reGGCodeArgs, match["args"]); argsMatch != nil {
			args = make(map[string]string)
			for i := range argsMatch["arg"] {
				arg := string(argsMatch["arg"][i])
				val := string(argsMatch["val"][i])
				args[arg] = val
			}
		}

		reader.Advance(len(restOfLine))
		return &ggcodeNode{
			Name: name,
			Args: args,
		}, parser.Continue | parser.HasChildren
	} else {
		return nil, parser.NoChildren
	}
}

func (s ggcodeBlockParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	line, _ := reader.PeekLine()
	if string(bytes.TrimSpace(line)) == "!!!" {
		reader.Advance(3)
		return parser.Close
	}
	return parser.Continue | parser.HasChildren
}

func (s ggcodeBlockParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {}

func (s ggcodeBlockParser) CanInterruptParagraph() bool {
	return false
}

func (s ggcodeBlockParser) CanAcceptIndentedLine() bool {
	return false
}

// Inline parser stuff

type ggcodeInlineParser struct{}

var _ parser.InlineParser = ggcodeInlineParser{}

func (s ggcodeInlineParser) Trigger() []byte {
	return []byte("!()")
}

func (s ggcodeInlineParser) Parse(parent gast.Node, block text.Reader, pc parser.Context) gast.Node {
	restOfLine, segment := block.PeekLine() // Gets the rest of the line (starting at the current parser cursor index), and the segment representing the indices in the source text.
	if match := extractMap(reGGCodeInline, restOfLine); match != nil {
		name := string(match["name"])
		var args map[string]string
		if argsMatch := extractAllMap(reGGCodeArgs, match["args"]); argsMatch != nil {
			args = make(map[string]string)
			for i := range argsMatch["arg"] {
				arg := string(argsMatch["arg"][i])
				val := string(argsMatch["val"][i])
				args[arg] = val
			}
		}

		node := &ggcodeNode{
			Name: name,
			Args: args,
		}
		contentLength := len(match["content"])
		if contentLength > 0 {
			contentSegmentStart := segment.Start + len(match["all"]) - (contentLength + 1) // the 1 is for the end parenthesis
			contentSegmentEnd := contentSegmentStart + contentLength
			contentSegment := text.NewSegment(contentSegmentStart, contentSegmentEnd)
			node.AppendChild(node, ast.NewTextSegment(contentSegment))
		}

		block.Advance(len(match["all"]))
		return node
	} else {
		return nil
	}
}

type ggcodeDelimiterParser struct {
	Node *ggcodeNode // We need to pass this through 🙄
}

func (p ggcodeDelimiterParser) IsDelimiter(b byte) bool {
	return b == '(' || b == ')'
}

func (p ggcodeDelimiterParser) CanOpenCloser(opener, closer *parser.Delimiter) bool {
	return opener.Char == '(' && closer.Char == ')'
}

func (p ggcodeDelimiterParser) OnMatch(consumes int) gast.Node {
	return p.Node
}

// ----------------------
// AST node
// ----------------------

type ggcodeNode struct {
	gast.BaseBlock
	Name string
	Args map[string]string
}

var _ ast.Node = &ggcodeNode{}

func (n *ggcodeNode) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, n.Args, nil)
}

var kindGGCode = gast.NewNodeKind("ggcode")

func (n *ggcodeNode) Kind() gast.NodeKind {
	return kindGGCode
}

// ----------------------
// Renderer
// ----------------------

type ggcodeHTMLRenderer struct {
	html.Config
	Opts MarkdownOptions
}

func newGGCodeHTMLRenderer(markdownOpts MarkdownOptions, opts ...html.Option) renderer.NodeRenderer {
	r := &ggcodeHTMLRenderer{
		Opts:   markdownOpts,
		Config: html.NewConfig(),
	}
	for _, opt := range opts {
		opt.SetHTMLOption(&r.Config)
	}
	return r
}

func (r *ggcodeHTMLRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(kindGGCode, r.render)
}

func (r *ggcodeHTMLRenderer) render(w util.BufWriter, source []byte, n gast.Node, entering bool) (gast.WalkStatus, error) {
	node := n.(*ggcodeNode)
	var renderer ggcodeRenderer = defaultGGCodeRenderer
	if tag, ok := ggcodeTags[node.Name]; ok {
		if tag.Filter == nil || tag.Filter(r.Opts) {
			renderer = tag.Renderer
		}
	}
	err := renderer(ggcodeRendererContext{
		W:      w,
		Source: source,
		Opts:   r.Opts,
	}, node, entering)
	return gast.WalkContinue, err
}

func defaultGGCodeRenderer(c ggcodeRendererContext, n *ggcodeNode, entering bool) error {
	if entering {
		c.W.WriteString("[unknown ggcode tag]")
	}
	return nil
}

// ----------------------
// Extension
// ----------------------

type ggcodeExtension struct {
	Opts MarkdownOptions
}

func (e ggcodeExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(parser.WithBlockParsers(
		util.Prioritized(ggcodeBlockParser{}, 500),
	))
	m.Parser().AddOptions(parser.WithInlineParsers(
		util.Prioritized(ggcodeInlineParser{}, 500),
	))
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(newGGCodeHTMLRenderer(e.Opts), 500),
	))
}
