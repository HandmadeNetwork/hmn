package main

import (
	"syscall/js"

	"git.handmade.network/hmn/hmn/src/parsing"
)

func main() {
	js.Global().Set("parseMarkdown", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return parsing.ParsePostInput(args[0].String(), parsing.PreviewMarkdown)
	}))

	var done chan bool
	<-done // block forever
}
