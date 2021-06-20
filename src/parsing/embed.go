package parsing

import (
	"regexp"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

var (
	REEmbedTag = regexp.MustCompile(`^!embed\((?P<url>.+?)\)`)

	// TODO: Timestamped youtube embeds
	REYoutubeLong  = regexp.MustCompile(`^https://www\.youtube\.com/watch?.*v=(?P<vid>[a-zA-Z0-9_-]{11})`)
	REYoutubeShort = regexp.MustCompile(`^https://youtu\.be/(?P<vid>[a-zA-Z0-9_-]{11})`)
	REVimeo        = regexp.MustCompile(`^https://vimeo\.com/(?P<vid>\d+)`)
)

// ----------------------
// Parser and delimiters
// ----------------------

type embedParser struct{}

func NewEmbedParser() parser.BlockParser {
	return embedParser{}
}

func (s embedParser) Trigger() []byte {
	return []byte{'!'}
}

func (s embedParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	restOfLine, _ := reader.PeekLine()
	urlMatch := REEmbedTag.FindSubmatch(restOfLine)
	if urlMatch == nil {
		return nil, parser.NoChildren
	}
	url := urlMatch[REEmbedTag.SubexpIndex("url")]

	html := ""
	if ytLongMatch := extract(REYoutubeLong, url, "vid"); ytLongMatch != nil {
		html = makeYoutubeEmbed(string(ytLongMatch))
	} else if ytShortMatch := extract(REYoutubeShort, url, "vid"); ytShortMatch != nil {
		html = makeYoutubeEmbed(string(ytShortMatch))
	} else if vimeoMatch := extract(REVimeo, url, "vid"); vimeoMatch != nil {
		html = `
<div class="mw6">
	<div class="aspect-ratio aspect-ratio--16x9">
		<iframe class="aspect-ratio--object" src="https://player.vimeo.com/video/` + string(vimeoMatch) + `" frameborder="0" allow="fullscreen; picture-in-picture" allowfullscreen></iframe>
	</div>
</div>`
	}

	if html == "" {
		return nil, parser.NoChildren
	}

	reader.Advance(len(urlMatch[0]))
	return NewEmbed(html), parser.NoChildren
}

func (s embedParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	return parser.Close
}

func (s embedParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {}

func (s embedParser) CanInterruptParagraph() bool {
	return true
}

func (s embedParser) CanAcceptIndentedLine() bool {
	return false
}

func extract(re *regexp.Regexp, src []byte, subexpName string) []byte {
	m := re.FindSubmatch(src)
	if m == nil {
		return nil
	}
	return m[re.SubexpIndex(subexpName)]
}

func makeYoutubeEmbed(vid string) string {
	return `
<div class="mw6">
	<div class="aspect-ratio aspect-ratio--16x9">
		<iframe class="aspect-ratio--object" src="https://www.youtube-nocookie.com/embed/` + vid + `" frameborder="0" allowfullscreen></iframe>
	</div>
</div>`
}

// ----------------------
// AST node
// ----------------------

type EmbedNode struct {
	gast.BaseBlock
	HTML string
}

func (n *EmbedNode) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, nil, nil)
}

var KindEmbed = gast.NewNodeKind("Embed")

func (n *EmbedNode) Kind() gast.NodeKind {
	return KindEmbed
}

func NewEmbed(HTML string) gast.Node {
	return &EmbedNode{
		HTML: HTML,
	}
}

// ----------------------
// Renderer
// ----------------------

type EmbedHTMLRenderer struct {
	html.Config
}

func NewEmbedHTMLRenderer(opts ...html.Option) renderer.NodeRenderer {
	r := &EmbedHTMLRenderer{
		Config: html.NewConfig(),
	}
	for _, opt := range opts {
		opt.SetHTMLOption(&r.Config)
	}
	return r
}

func (r *EmbedHTMLRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindEmbed, r.renderEmbed)
}

func (r *EmbedHTMLRenderer) renderEmbed(w util.BufWriter, source []byte, n gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		w.WriteString(n.(*EmbedNode).HTML)
	}
	return gast.WalkSkipChildren, nil
}

// ----------------------
// Extension
// ----------------------

type EmbedExtension struct{}

func (e EmbedExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(parser.WithBlockParsers(
		util.Prioritized(NewEmbedParser(), 500),
	))
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(NewEmbedHTMLRenderer(), 500),
	))
}
