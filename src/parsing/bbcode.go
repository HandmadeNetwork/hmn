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

var previewBBCodeCompiler = bbcode.NewCompiler(false, false)
var realBBCodeCompiler = bbcode.NewCompiler(false, false)

var REYoutubeVidOnly = regexp.MustCompile(`^[a-zA-Z0-9_-]{11}$`)

func init() {
	previewBBCodeCompiler.SetTag("youtube", makeYoutubeBBCodeFunc(true))
	realBBCodeCompiler.SetTag("youtube", makeYoutubeBBCodeFunc(false))
}

func makeYoutubeBBCodeFunc(preview bool) bbcode.TagCompilerFunc {
	return func(bn *bbcode.BBCodeNode) (*bbcode.HTMLTag, bool) {
		if len(bn.Children) != 1 {
			return bbcode.NewHTMLTag("<missing video URL>"), false
		}
		if bn.Children[0].Token.ID != bbcode.TEXT {
			return bbcode.NewHTMLTag("<missing video URL>"), false
		}

		contents := bn.Children[0].Token.Value.(string)
		vid := ""

		if m := REYoutubeLong.FindStringSubmatch(contents); m != nil {
			vid = m[REYoutubeLong.SubexpIndex("vid")]
		} else if m := REYoutubeShort.FindStringSubmatch(contents); m != nil {
			vid = m[REYoutubeShort.SubexpIndex("vid")]
		} else if m := REYoutubeVidOnly.MatchString(contents); m {
			vid = contents
		}

		if vid == "" {
			return bbcode.NewHTMLTag("<bad video URL>"), false
		}

		if preview {
			/*
				<div class="mw6">
					<img src="https://img.youtube.com/vi/` + vid + `/hqdefault.jpg">
				</div>
			*/

			out := bbcode.NewHTMLTag("")
			out.Name = "div"
			out.Attrs["class"] = "mw6"

			img := bbcode.NewHTMLTag("")
			img.Name = "img"
			img.Attrs = map[string]string{
				"src": "https://img.youtube.com/vi/" + vid + "/hqdefault.jpg",
			}

			out.AppendChild(img)

			return out, false
		} else {
			/*
				<div class="mw6">
					<div class="aspect-ratio aspect-ratio--16x9">
						<iframe class="aspect-ratio--object" src="https://www.youtube-nocookie.com/embed/` + vid + `" frameborder="0" allowfullscreen></iframe>
					</div>
				</div>
			*/

			out := bbcode.NewHTMLTag("")
			out.Name = "div"
			out.Attrs["class"] = "mw6"

			aspect := bbcode.NewHTMLTag("")
			aspect.Name = "div"
			aspect.Attrs["class"] = "aspect-ratio aspect-ratio--16x9"

			iframe := bbcode.NewHTMLTag("")
			iframe.Name = "iframe"
			iframe.Attrs = map[string]string{
				"class":           "aspect-ratio--object",
				"src":             "https://www.youtube-nocookie.com/embed/" + vid,
				"frameborder":     "0",
				"allowfullscreen": "",
			}
			iframe.AppendChild(nil) // render a closing tag lol

			aspect.AppendChild(iframe)
			out.AppendChild(aspect)

			return out, false
		}
	}
}

// ----------------------
// Parser and delimiters
// ----------------------

type bbcodeParser struct {
	Preview bool
}

var _ parser.InlineParser = &bbcodeParser{}

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
					break
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

	compiler := realBBCodeCompiler
	if s.Preview {
		compiler = previewBBCodeCompiler
	}

	return NewBBCode(compiler.Compile(string(unparsedBBCode)))
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

type BBCodeExtension struct {
	Preview bool
}

func (e BBCodeExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(parser.WithInlineParsers(
		util.Prioritized(bbcodeParser{Preview: e.Preview}, BBCodePriority),
	))
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(NewBBCodeHTMLRenderer(), BBCodePriority),
	))
}
