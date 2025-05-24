package main

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"time"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: go run timestamps.go <fishbowl>.html <fishbowl>-timestamped.html")
		os.Exit(1)
	}

	htmlPath := os.Args[1]
	htmlOutPath := os.Args[2]

	htmlBytes, err := os.ReadFile(htmlPath)
	if err != nil {
		panic(err)
	}

	html := string(htmlBytes)

	regex := regexp.MustCompile(
		`(<span class="?chatlog__timestamp"?><a href=[^>]+>)(\d+)/(\d+)/(\d+)( [^<]+</a></span>)`,
	)

	htmlOut := regex.ReplaceAllStringFunc(html, func(s string) string {
		match := regex.FindStringSubmatch(s)
		month, err := strconv.ParseInt(match[2], 10, 64)
		if err != nil {
			panic(err)
		}
		monthStr := time.Month(month).String()
		return fmt.Sprintf("%s%s %s, %s%s", match[1], monthStr, match[3], match[4], match[5])
	})

	err = os.WriteFile(htmlOutPath, []byte(htmlOut), 0666)
	if err != nil {
		panic(err)
	}
}
