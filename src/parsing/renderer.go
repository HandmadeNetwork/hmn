package parsing

import (
	"io"
	"regexp"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
)

type plaintextRenderer struct{}

var _ renderer.Renderer = plaintextRenderer{}

var backslashRegex = regexp.MustCompile("\\\\(?P<char>[\\\\\\x60!\"#$%&'()*+,-./:;<=>?@\\[\\]^_{|}~])")

func (r plaintextRenderer) Render(w io.Writer, source []byte, n ast.Node) error {
	return ast.Walk(n, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch n.Kind() {
		case ast.KindText:
			n := n.(*ast.Text)
			_, err := w.Write(backslashRegex.ReplaceAll(n.Text(source), []byte("$1")))
			if err != nil {
				return ast.WalkContinue, err
			}

			if n.SoftLineBreak() {
				_, err := w.Write([]byte(" "))
				if err != nil {
					return ast.WalkContinue, err
				}
			}
		case ast.KindParagraph:
			_, err := w.Write([]byte(" "))
			if err != nil {
				return ast.WalkContinue, err
			}
		}

		return ast.WalkContinue, nil
	})
}

func (r plaintextRenderer) AddOptions(...renderer.Option) {}
