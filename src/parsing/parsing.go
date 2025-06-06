package parsing

import (
	"bytes"

	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"

	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/util"
)

// Used for rendering real-time previews of post content.
var ForumPreviewMarkdown = makeGoldmark(
	false,
	goldmark.WithExtensions(makeGoldmarkExtensions(MarkdownOptions{
		Previews: true,
		Embeds:   true,
	})...),
)

// Used for generating the final HTML for a post.
var ForumRealMarkdown = makeGoldmark(
	false,
	goldmark.WithExtensions(makeGoldmarkExtensions(MarkdownOptions{
		Previews: false,
		Embeds:   true,
	})...),
)

// Used for generating plain-text previews of posts.
var PlaintextMarkdown = makeGoldmark(
	false,
	goldmark.WithExtensions(makeGoldmarkExtensions(MarkdownOptions{
		Previews: false,
		Embeds:   true,
	})...),
	goldmark.WithRenderer(plaintextRenderer{}),
)

// Used for processing Discord messages
var DiscordMarkdown = makeGoldmark(
	false,
	goldmark.WithExtensions(makeGoldmarkExtensions(MarkdownOptions{
		Previews: false,
		Embeds:   false,
	})...),
	goldmark.WithRendererOptions(html.WithHardWraps()),
)

var DiscordTagMarkdown = makeGoldmark(
	false,
	goldmark.WithExtensions(makeGoldmarkExtensions(MarkdownOptions{
		Previews: false,
		Embeds:   false,
	})...),
	goldmark.WithRendererOptions(html.WithHardWraps()),
	goldmark.WithRenderer(projectTagRenderer{}),
)

// Used for rendering real-time previews of post content.
var EducationPreviewMarkdown = makeGoldmark(
	true,
	goldmark.WithExtensions(makeGoldmarkExtensions(MarkdownOptions{
		Previews:  true,
		Embeds:    true,
		Education: true,
	})...),
	goldmark.WithRendererOptions(html.WithUnsafe()),
)

// Used for generating the final HTML for a post.
var EducationRealMarkdown = makeGoldmark(
	true,
	goldmark.WithExtensions(makeGoldmarkExtensions(MarkdownOptions{
		Previews:  false,
		Embeds:    true,
		Education: true,
	})...),
	goldmark.WithRendererOptions(html.WithUnsafe()),
)

func ParseMarkdown(source string, md goldmark.Markdown) string {
	var buf bytes.Buffer
	if err := md.Convert([]byte(source), &buf); err != nil {
		panic(err)
	}

	return buf.String()
}

type MarkdownOptions struct {
	Previews  bool
	Embeds    bool
	Education bool
}

func makeGoldmark(rawHTML bool, opts ...goldmark.Option) goldmark.Markdown {
	// We need to re-create Goldmark's default parsers to disable HTML parsing.
	// Or enable it again. yay

	// See parser.DefaultBlockParsers
	blockParsers := []util.PrioritizedValue{
		util.Prioritized(parser.NewSetextHeadingParser(), 100),
		util.Prioritized(parser.NewThematicBreakParser(), 200),
		util.Prioritized(parser.NewListParser(), 300),
		util.Prioritized(parser.NewListItemParser(), 400),
		util.Prioritized(parser.NewCodeBlockParser(), 500),
		util.Prioritized(parser.NewATXHeadingParser(), 600),
		util.Prioritized(parser.NewFencedCodeBlockParser(), 700),
		util.Prioritized(parser.NewBlockquoteParser(), 800),
		//util.Prioritized(parser.NewHTMLBlockParser(), 900),
		util.Prioritized(parser.NewParagraphParser(), 1000),
	}

	// See parser.DefaultInlineParsers
	inlineParsers := []util.PrioritizedValue{
		util.Prioritized(parser.NewCodeSpanParser(), 100),
		util.Prioritized(parser.NewLinkParser(), 200),
		util.Prioritized(parser.NewAutoLinkParser(), 300),
		// util.Prioritized(parser.NewRawHTMLParser(), 400),
		util.Prioritized(parser.NewEmphasisParser(), 500),
	}

	if rawHTML {
		blockParsers = append(blockParsers, util.Prioritized(parser.NewHTMLBlockParser(), 900))
		inlineParsers = append(inlineParsers, util.Prioritized(parser.NewRawHTMLParser(), 400))
	}

	opts = append(opts, goldmark.WithParser(parser.NewParser(
		parser.WithBlockParsers(blockParsers...),
		parser.WithInlineParsers(inlineParsers...),
		parser.WithParagraphTransformers(parser.DefaultParagraphTransformers()...),
	)))

	return goldmark.New(opts...)
}

func makeGoldmarkExtensions(opts MarkdownOptions) []goldmark.Extender {
	var extenders []goldmark.Extender
	extenders = append(extenders,
		extension.GFM,
		highlightExtension,
		SpoilerExtension{},
	)

	if opts.Embeds {
		extenders = append(extenders,
			EmbedExtension{
				Preview: opts.Previews,
			},
		)
	}

	extenders = append(extenders,
		MathjaxExtension{},
		BBCodeExtension{
			Preview: opts.Previews,
		},
		ggcodeExtension{
			Opts: opts,
		},
	)

	return extenders
}

var highlightExtension = highlighting.NewHighlighting(
	highlighting.WithFormatOptions(HMNChromaOptions...),
	highlighting.WithWrapperRenderer(func(w util.BufWriter, context highlighting.CodeBlockContext, entering bool) {
		if entering {
			w.WriteString(`<pre class="hmn-code">`)
		} else {
			w.WriteString(`</pre>`)
		}
	}),
)
