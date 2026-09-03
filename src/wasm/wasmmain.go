//go:build js

package main

import (
	"syscall/js"

	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/links"
	"git.handmade.network/hmn/hmn/src/parsing"
)

// NOTE(ben): We do not include the config package because it includes pgx and
// zerolog, which we absolutely do not need in wasm. But this means that we
// don't actually know what some of the URL-related config is, but also, this
// is fine because we want to ship the same wasm blob to both beta and live
// anyway, at least for now. So here we just do the same things that would be
// done in config/init.go, with default values, and if there are some broken
// links in the Markdown previews in beta and dev, we don't care.
func init() {
	hmnurl.SetGlobalBaseUrl("https://handmade.network/")
	hmnurl.SetS3BaseUrl("https://assets.media.handmade.network/")
}

// Build using `hmn bundle wasm`.
func main() {
	js.Global().Set("parseMarkdown", js.FuncOf(func(this js.Value, args []js.Value) any {
		return parsing.ParseMarkdown(args[0].String(), parsing.PostPreviewMarkdown)
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
