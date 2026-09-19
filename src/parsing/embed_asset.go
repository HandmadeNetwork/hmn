package parsing

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

var (
	// NOTE(ben): If you update this regex, make sure to also update the file
	// extensions recognized in markdown_upload.ts to ensure that uploaded files
	// get the ![foo](url) syntax.
	PlainVideoLinkOnly = `https?://(handmade\.network|beta-handmadenet\.work|([^/.:]+\.)?handmade\.local(:\d+)?|assets\.media\.handmade\.network|hmn-assets-2\.ams3\.cdn\.digitaloceanspaces\.com|localhost(:\d+)?)/[^\s"'<>()]+?\.(mp4|webm|mov|m4v)`

	REPlainVideoBareLink     = regexp.MustCompile(`(?i)^` + PlainVideoLinkOnly + `$`)
	REPlainVideoMarkdownLink = regexp.MustCompile(`(?i)^!\[[^\]]*?\]\((?P<url>` + PlainVideoLinkOnly + `)\)$`)
)

func videoEmbed(url string, preview bool) string {
	if preview {
		return `
			<div class="aspect-ratio-real--16x9 ba bg1 i flex items-center justify-center">
				Embedded video
			</div>
		`
	} else {
		return fmt.Sprintf(`<video src="%s" preload="metadata" controls></video>`, url)
	}
}

func htmlForAssetEmbed(url string, preview bool) (string, []byte, bool) {
	url = strings.TrimSpace(url)

	if match := REPlainVideoMarkdownLink.FindSubmatch([]byte(url)); match != nil {
		url := match[REPlainVideoMarkdownLink.SubexpIndex("url")]
		return videoEmbed(string(url), preview), match[0], true
	} else if match := REPlainVideoBareLink.FindString(url); match != "" {
		return videoEmbed(match, preview), []byte(match), true
	}

	return "", nil, false
}

// ----------------------
// Parser and delimiters
// ----------------------

type assetEmbedParser struct {
	Preview bool
}

var _ parser.BlockParser = assetEmbedParser{}

func (s assetEmbedParser) Trigger() []byte {
	return nil
}

func (s assetEmbedParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	restOfLine, _ := reader.PeekLine()

	if html, match, ok := htmlForAssetEmbed(string(restOfLine), s.Preview); ok {
		reader.Advance(len(match))
		return NewAssetEmbed(html), parser.NoChildren
	} else {
		return nil, parser.NoChildren
	}
}

func (s assetEmbedParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	return parser.Close
}

func (s assetEmbedParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {}

func (s assetEmbedParser) CanInterruptParagraph() bool {
	return true
}

func (s assetEmbedParser) CanAcceptIndentedLine() bool {
	return false
}

// ----------------------
// AST node
// ----------------------

type AssetEmbedNode struct {
	ast.BaseBlock
	HTML string
}

func (n *AssetEmbedNode) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, nil, nil)
}

var KindAssetEmbed = ast.NewNodeKind("AssetEmbed")

func (n *AssetEmbedNode) Kind() ast.NodeKind {
	return KindAssetEmbed
}

func NewAssetEmbed(HTML string) ast.Node {
	return &AssetEmbedNode{
		HTML: HTML,
	}
}

// ----------------------
// Renderer
// ----------------------

type AssetEmbedHTMLRenderer struct {
	html.Config
}

func NewAssetEmbedHTMLRenderer(opts ...html.Option) renderer.NodeRenderer {
	r := &AssetEmbedHTMLRenderer{
		Config: html.NewConfig(),
	}
	for _, opt := range opts {
		opt.SetHTMLOption(&r.Config)
	}
	return r
}

func (r *AssetEmbedHTMLRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindAssetEmbed, r.renderAssetEmbed)
}

func (r *AssetEmbedHTMLRenderer) renderAssetEmbed(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		w.WriteString(n.(*AssetEmbedNode).HTML)
	}
	return ast.WalkSkipChildren, nil
}

// ----------------------
// Extension
// ----------------------

type AssetEmbedExtension struct {
	Preview bool
}

func (e AssetEmbedExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(parser.WithBlockParsers(
		util.Prioritized(assetEmbedParser{Preview: e.Preview}, 500),
	))
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(NewAssetEmbedHTMLRenderer(), 500),
	))
}
