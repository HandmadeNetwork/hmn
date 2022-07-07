package main

import (
	"fmt"
	"os"
	"regexp"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: go run timestamps.go [fishbowl].html [fishbowl]-timestamped.html")
		os.Exit(1)
	}

	htmlPath := os.Args[1]
	htmlOutPath := os.Args[2]

	htmlBytes, err := os.ReadFile(htmlPath)
	if err != nil {
		panic(err)
	}

	html := string(htmlBytes)

	regex, err := regexp.Compile(
		"(<span class=\"chatlog__timestamp\">)(\\d+)-([A-Za-z]+)-(\\d+)( [^<]+</span>)",
	)
	if err != nil {
		panic(err)
	}

	htmlOut := regex.ReplaceAllString(
		html,
		"$1$3 $2, 20$4$5",
	)

	err = os.WriteFile(htmlOutPath, []byte(htmlOut), 0666)
	if err != nil {
		panic(err)
	}
}
