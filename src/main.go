package main

import (
	_ "git.handmade.network/hmn/hmn/src/migration"
	"git.handmade.network/hmn/hmn/src/website"
)

func main() {
	website.WebsiteCommand.Execute()
}
