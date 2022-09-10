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
	// TODO: Timestamped youtube embeds
	REYoutubeLong  = regexp.MustCompile(`^https://www\.youtube\.com/watch?.*v=(?P<vid>[a-zA-Z0-9_-]{11})`)
	REYoutubeShort = regexp.MustCompile(`^https://youtu\.be/(?P<vid>[a-zA-Z0-9_-]{11})`)
	REVimeo        = regexp.MustCompile(`^https://vimeo\.com/(?P<vid>\d+)`)
	// TODO: Twitch VODs / clips
	// TODO: Desmos
	// TODO: Tweets
)

// ----------------------
// Parser and delimiters
// ----------------------

type embedParser struct {
	Preview bool
}

var _ parser.BlockParser = embedParser{}

func (s embedParser) Trigger() []byte {
	return nil
}

func (s embedParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	restOfLine, _ := reader.PeekLine()

	html := ""
	var match []byte
	if ytLongMatch := extract(REYoutubeLong, restOfLine, "vid"); ytLongMatch != nil {
		match = ytLongMatch
		html = makeYoutubeEmbed(string(ytLongMatch), s.Preview)
	} else if ytShortMatch := extract(REYoutubeShort, restOfLine, "vid"); ytShortMatch != nil {
		match = ytShortMatch
		html = makeYoutubeEmbed(string(ytShortMatch), s.Preview)
	} else if vimeoMatch := extract(REVimeo, restOfLine, "vid"); vimeoMatch != nil {
		match = vimeoMatch
		html = s.previewOrLegitEmbed("Vimeo", `
<div class="mw6">
	<div class="aspect-ratio aspect-ratio--16x9">
		<iframe class="aspect-ratio--object" src="https://player.vimeo.com/video/`+string(vimeoMatch)+`" frameborder="0" allow="fullscreen; picture-in-picture" allowfullscreen></iframe>
	</div>
</div>`)
	}

	if html == "" {
		return nil, parser.NoChildren
	}

	reader.Advance(len(match))
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

func (s embedParser) previewOrLegitEmbed(name string, legitHtml string) string {
	if s.Preview {
		return `
<div class="mw6">
	<div class="aspect-ratio aspect-ratio--16x9">
		<div class="aspect-ratio--object ba b--dimmest bg-light-gray i black flex items-center justify-center">
			` + name + ` embed
		</div>
	</div>
</div>
`
	}

	return legitHtml
}

func makeYoutubeEmbed(vid string, preview bool) string {
	if preview {
		return `
<div class="mw6">
	<img src="https://img.youtube.com/vi/` + vid + `/hqdefault.jpg">
</div>
`
	} else {
		return `
<div class="mw6">
	<div class="aspect-ratio aspect-ratio--16x9">
		<iframe class="aspect-ratio--object" src="https://www.youtube-nocookie.com/embed/` + vid + `" frameborder="0" allowfullscreen></iframe>
	</div>
</div>
`
	}
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

type EmbedExtension struct {
	Preview bool
}

func (e EmbedExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(parser.WithBlockParsers(
		util.Prioritized(embedParser{Preview: e.Preview}, 500),
	))
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(NewEmbedHTMLRenderer(), 500),
	))
}
