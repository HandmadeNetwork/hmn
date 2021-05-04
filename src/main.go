package main

import (
	_ "git.handmade.network/hmn/hmn/src/admintools"
	_ "git.handmade.network/hmn/hmn/src/buildscss"
	_ "git.handmade.network/hmn/hmn/src/migration"
	"git.handmade.network/hmn/hmn/src/website"
)

func main() {
	website.WebsiteCommand.Execute()
}
