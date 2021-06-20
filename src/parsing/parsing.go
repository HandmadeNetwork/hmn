package parsing

import (
	"bytes"
	"regexp"
	"sort"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

func StripNames(regexStr string) string {
	return regexp.MustCompile(`\(\?P<[a-zA-Z0-9_]+>`).ReplaceAllString(regexStr, `(?:`)
}

var reArgStr = `(?P<name>[a-zA-Z0-9]+)(?:\s*=\s*(?:'(?P<single_quoted_val>.*?)'|"(?P<double_quoted_val>.*?)"|(?P<bare_val>[^\s\]]+)))?`
var reTagOpenStr = `\[\s*(?P<args>(?:` + StripNames(reArgStr) + `)(?:\s+(?:` + StripNames(reArgStr) + `))*)\s*\]`
var reTagCloseStr = `\[/\s*(?P<name>[a-zA-Z0-9]+)?\s*\]`

var reArg = regexp.MustCompile(reArgStr)
var reTagOpen = regexp.MustCompile(reTagOpenStr)
var reTagClose = regexp.MustCompile(reTagCloseStr)

const tokenTypeString = "string"
const tokenTypeOpenTag = "openTag"
const tokenTypeCloseTag = "closeTag"

type token struct {
	Type       string
	StartIndex int
	EndIndex   int
	Contents   string
}

func ParseBBCode(input string) string {
	return input
}

func tokenizeBBCode(input string) []token {
	openMatches := reTagOpen.FindAllStringIndex(input, -1)
	closeMatches := reTagClose.FindAllStringIndex(input, -1)

	// Build tokens from regex matches
	var tagTokens []token
	for _, match := range openMatches {
		tagTokens = append(tagTokens, token{
			Type:       tokenTypeOpenTag,
			StartIndex: match[0],
			EndIndex:   match[1],
			Contents:   input[match[0]:match[1]],
		})
	}
	for _, match := range closeMatches {
		tagTokens = append(tagTokens, token{
			Type:       tokenTypeCloseTag,
			StartIndex: match[0],
			EndIndex:   match[1],
			Contents:   input[match[0]:match[1]],
		})
	}

	// Sort those tokens together
	sort.Slice(tagTokens, func(i, j int) bool {
		return tagTokens[i].StartIndex < tagTokens[j].StartIndex
	})

	// Make a new list of tokens that fills in the gaps with plain old text
	var tokens []token
	for i, tagToken := range tagTokens {
		prevEnd := 0
		if i > 0 {
			prevEnd = tagTokens[i-1].EndIndex
		}

		tokens = append(tokens, token{
			Type:       tokenTypeString,
			StartIndex: prevEnd,
			EndIndex:   tagToken.StartIndex,
			Contents:   input[prevEnd:tagToken.StartIndex],
		})
		tokens = append(tokens, tagToken)
	}
	tokens = append(tokens, token{
		Type:       tokenTypeString,
		StartIndex: tokens[len(tokens)-1].EndIndex,
		EndIndex:   len(input),
		Contents:   input[tokens[len(tokens)-1].EndIndex:],
	})

	return tokens
}

var previewMarkdown = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,
		SpoilerExtension{},
		EmbedExtension{
			Preview: true,
		},
		bTag{},
	),
)
var realMarkdown = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,
		SpoilerExtension{},
		EmbedExtension{},
		bTag{},
	),
)

func ParsePostInput(source string, preview bool) string {
	md := realMarkdown
	if preview {
		md = previewMarkdown
	}

	var buf bytes.Buffer
	if err := md.Convert([]byte(source), &buf); err != nil {
		panic(err)
	}

	return buf.String()
}
