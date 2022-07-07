package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Println("Usage: go run twemoji.go [fishbowl].html files [fishbowl]-twemojied.html")
		os.Exit(1)
	}

	htmlPath := os.Args[1]
	filesDir := os.Args[2]
	htmlOutPath := os.Args[3]

	htmlBytes, err := os.ReadFile(htmlPath)
	if err != nil {
		panic(err)
	}

	html := string(htmlBytes)

	for {
		linkStart := strings.Index(html, "https://twemoji.maxcdn.com/")
		if linkStart == -1 {
			break
		}

		linkEnd := strings.Index(html[linkStart:], "\"") + linkStart
		link := html[linkStart:linkEnd]
		emojiFilenameStart := strings.LastIndex(link, "/") + 1
		emojiFilename := "twemoji_" + link[emojiFilenameStart:]
		emojiPath := path.Join(filesDir, emojiFilename)

		emojiResponse, err := http.Get(link)
		if err != nil {
			panic(err)
		}
		defer emojiResponse.Body.Close()

		if emojiResponse.StatusCode > 299 {
			panic("Non-200 status code: " + fmt.Sprint(emojiResponse.StatusCode))
		}

		emojiFile, err := os.Create(emojiPath)
		if err != nil {
			panic(err)
		}
		defer emojiFile.Close()

		_, err = io.Copy(emojiFile, emojiResponse.Body)
		if err != nil {
			panic(err)
		}

		html = strings.ReplaceAll(html, link, emojiPath)

		fmt.Println(emojiFilename)
	}

	html = strings.ReplaceAll(
		html,
		"<div class=\"chatlog\">",
		"<div class=\"chatlog\">\n<!-- Emojis by Twitter's Twemoji https://twemoji.twitter.com/ -->",
	)

	err = os.WriteFile(htmlOutPath, []byte(html), 0666)
	if err != nil {
		panic(err)
	}
}
