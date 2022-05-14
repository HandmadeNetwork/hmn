package main

import (
	_ "git.handmade.network/hmn/hmn/src/admintools"
	_ "git.handmade.network/hmn/hmn/src/hmns3"
	_ "git.handmade.network/hmn/hmn/src/assets"
	_ "git.handmade.network/hmn/hmn/src/buildscss"
	_ "git.handmade.network/hmn/hmn/src/discord/cmd"
	_ "git.handmade.network/hmn/hmn/src/initimage"
	_ "git.handmade.network/hmn/hmn/src/migration"
	"git.handmade.network/hmn/hmn/src/website"
)

func main() {
	website.WebsiteCommand.Execute()
}
