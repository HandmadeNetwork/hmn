package parsing

import (
	"regexp"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
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

// Returns the HTML used to embed the given url, or false if the media cannot be embedded.
//
// The provided string need only start with the URL; if there is trailing content after the URL, it
// will be ignored.
func htmlForURLEmbed(url string, preview bool) (string, []byte, bool) {
	if match := extract(REYoutubeLong, []byte(url), "vid"); match != nil {
		return makeYoutubeEmbed(string(match), preview), match, true
	} else if match := extract(REYoutubeShort, []byte(url), "vid"); match != nil {
		return makeYoutubeEmbed(string(match), preview), match, true
	} else if match := extract(REVimeo, []byte(url), "vid"); match != nil {
		return previewOrLegitExternalEmbed("Vimeo", `
			<div class="aspect-ratio aspect-ratio--16x9">
				<iframe class="aspect-ratio--object" src="https://player.vimeo.com/video/`+string(match)+`" frameborder="0" allow="fullscreen; picture-in-picture" allowfullscreen></iframe>
			</div>
		`, preview), match, true
	}

	return "", nil, false
}

// ----------------------
// Parser and delimiters
// ----------------------

type externalEmbedParser struct {
	Preview bool
}

var _ parser.BlockParser = externalEmbedParser{}

func (s externalEmbedParser) Trigger() []byte {
	return nil
}

func (s externalEmbedParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	restOfLine, _ := reader.PeekLine()

	if html, match, ok := htmlForURLEmbed(string(restOfLine), s.Preview); ok {
		html = `<div class="mw6">` + html + `</div>`
		reader.Advance(len(match))
		return NewExternalEmbed(html), parser.NoChildren
	} else {
		return nil, parser.NoChildren
	}
}

func (s externalEmbedParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	return parser.Close
}

func (s externalEmbedParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {}

func (s externalEmbedParser) CanInterruptParagraph() bool {
	return true
}

func (s externalEmbedParser) CanAcceptIndentedLine() bool {
	return false
}

func previewOrLegitExternalEmbed(name, legitHTML string, preview bool) string {
	if preview {
		return `
			<div class="aspect-ratio aspect-ratio--16x9">
				<div class="aspect-ratio--object ba b--dimmest bg-light-gray i black flex items-center justify-center">
					` + name + ` embed
				</div>
			</div>
		`
	} else {
		return legitHTML
	}
}

func makeYoutubeEmbed(vid string, preview bool) string {
	if preview {
		return `<img src="https://img.youtube.com/vi/` + vid + `/hqdefault.jpg">`
	} else {
		return `
			<div class="aspect-ratio aspect-ratio--16x9">
				<iframe class="aspect-ratio--object" src="https://www.youtube-nocookie.com/embed/` + vid + `" frameborder="0" allowfullscreen></iframe>
			</div>
		`
	}
}

// ----------------------
// AST node
// ----------------------

type ExternalEmbedNode struct {
	ast.BaseBlock
	HTML string
}

func (n *ExternalEmbedNode) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, nil, nil)
}

var KindExternalEmbed = ast.NewNodeKind("ExternalEmbed")

func (n *ExternalEmbedNode) Kind() ast.NodeKind {
	return KindExternalEmbed
}

func NewExternalEmbed(HTML string) ast.Node {
	return &ExternalEmbedNode{
		HTML: HTML,
	}
}

// ----------------------
// Renderer
// ----------------------

type ExternalEmbedHTMLRenderer struct {
	html.Config
}

func NewExternalEmbedHTMLRenderer(opts ...html.Option) renderer.NodeRenderer {
	r := &ExternalEmbedHTMLRenderer{
		Config: html.NewConfig(),
	}
	for _, opt := range opts {
		opt.SetHTMLOption(&r.Config)
	}
	return r
}

func (r *ExternalEmbedHTMLRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindExternalEmbed, r.renderExternalEmbed)
}

func (r *ExternalEmbedHTMLRenderer) renderExternalEmbed(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		w.WriteString(n.(*ExternalEmbedNode).HTML)
	}
	return ast.WalkSkipChildren, nil
}

// ----------------------
// Extension
// ----------------------

type ExternalEmbedExtension struct {
	Preview bool
}

func (e ExternalEmbedExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(parser.WithBlockParsers(
		util.Prioritized(externalEmbedParser{Preview: e.Preview}, 500),
	))
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(NewExternalEmbedHTMLRenderer(), 500),
	))
}
