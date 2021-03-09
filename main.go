package main

import (
	_ "git.handmade.network/hmn/hmn/migration"
	"git.handmade.network/hmn/hmn/website"
)

func main() {
	website.WebsiteCommand.Execute()
}
