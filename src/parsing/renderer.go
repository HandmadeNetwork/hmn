package parsing

import (
	"io"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
)

type plaintextRenderer struct{}

var _ renderer.Renderer = plaintextRenderer{}

func (r plaintextRenderer) Render(w io.Writer, source []byte, n ast.Node) error {
	return ast.Walk(n, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch n.Kind() {
		case ast.KindText:
			n := n.(*ast.Text)

			_, err := w.Write(n.Text(source))
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
