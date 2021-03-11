package color

import "runtime"

// See this file for a good color reference:
// https://github.com/fatih/color/blob/master/color.go

var Reset = "\033[0m"
var Bold = "\033[1m"
var Faint = "\033[2m"
var Italic = "\033[3m"
var Underline = "\033[4m"
var BlinkSlow = "\033[5m"
var BlinkRapid = "\033[6m"
var ReverseVideo = "\033[7m"
var Concealed = "\033[8m"
var CrossedOut = "\033[9m"

var Red = "\033[31m"
var Green = "\033[32m"
var Yellow = "\033[33m"
var Blue = "\033[34m"
var Purple = "\033[35m"
var Cyan = "\033[36m"
var Gray = "\033[37m"
var White = "\033[97m"

var BgBlack = "\033[40m"
var BgRed = "\033[41m"
var BgGreen = "\033[42m"
var BgYellow = "\033[43m"
var BgBlue = "\033[44m"
var BgMagenta = "\033[45m"
var BgCyan = "\033[46m"
var BgWhite = "\033[47m"

func init() {
	if runtime.GOOS == "windows" {
		Reset = ""
		Red = ""
		Green = ""
		Yellow = ""
		Blue = ""
		Purple = ""
		Cyan = ""
		Gray = ""
		White = ""
	}
}
