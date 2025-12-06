//go:build js

package main

import (
	"syscall/js"

	"git.handmade.network/hmn/hmn/src/links"
	"git.handmade.network/hmn/hmn/src/parsing"
)

func main() {
	js.Global().Set("parseMarkdown", js.FuncOf(func(this js.Value, args []js.Value) any {
		return parsing.ParseMarkdown(args[0].String(), parsing.ForumPreviewMarkdown)
	}))
	js.Global().Set("parseMarkdownEdu", js.FuncOf(func(this js.Value, args []js.Value) any {
		return parsing.ParseMarkdown(args[0].String(), parsing.EducationPreviewMarkdown)
	}))
	js.Global().Set("parseKnownServicesForUrl", js.FuncOf(func(this js.Value, args []js.Value) any {
		service, username := links.ParseKnownServicesForUrl(args[0].String())
		return js.ValueOf(map[string]any{
			"service":  service.Name,
			"icon":     service.IconName,
			"username": username,
		})
	}))

	var done chan struct{}
	<-done // block forever
}
