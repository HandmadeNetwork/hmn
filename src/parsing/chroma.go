package parsing

import "github.com/alecthomas/chroma/formatters/html"

var HMNChromaOptions = []html.Option{
	html.WithClasses(true),
	html.WithPreWrapper(nopPreWrapper{}),
}

type nopPreWrapper struct{}

var _ html.PreWrapper = nopPreWrapper{}

func (w nopPreWrapper) Start(code bool, styleAttr string) string {
	return ""
}

func (w nopPreWrapper) End(code bool) string {
	return ""
}
