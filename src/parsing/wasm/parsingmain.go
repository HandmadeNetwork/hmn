//go:build js

package main

import (
	"syscall/js"

	"git.handmade.network/hmn/hmn/src/parsing"
)

func main() {
	js.Global().Set("parseMarkdown", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return parsing.ParseMarkdown(args[0].String(), parsing.ForumPreviewMarkdown)
	}))
	js.Global().Set("parseMarkdownEdu", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return parsing.ParseMarkdown(args[0].String(), parsing.EducationPreviewMarkdown)
	}))

	var done chan bool
	<-done // block forever
}
