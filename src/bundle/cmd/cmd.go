package cmd

import (
	"fmt"
	"os"
	"strings"
	"unicode"

	color "git.handmade.network/hmn/hmn/src/ansicolor"
	"git.handmade.network/hmn/hmn/src/bundle"
	"git.handmade.network/hmn/hmn/src/website"
	"github.com/evanw/esbuild/pkg/api"
	"github.com/spf13/cobra"
)

func init() {
	buildCommand := &cobra.Command{
		Use:   "bundle",
		Short: "Build the website CSS and JS. (Normally this happens automatically during development.)",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, err := bundle.BuildContext()
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			res := ctx.Rebuild()
			for _, warning := range res.Warnings {
				printEsBuildMessage(warning, false)
			}
			for _, err := range res.Errors {
				printEsBuildMessage(err, true)
			}

			if len(res.Errors) > 0 {
				os.Exit(1)
			}
		},
	}
	website.WebsiteCommand.AddCommand(buildCommand)
}

func printEsBuildMessage(msg api.Message, isError bool) {
	fgColor := color.Yellow
	bgColor := color.BgYellow
	upperName := "WARNING"
	lowerName := "warning"
	if isError {
		fgColor = color.Red
		bgColor = color.BgRed
		upperName = "ERROR"
		lowerName = "error"
	}

	code := ""
	if msg.ID != "" {
		code = " (" + msg.ID + ")"
	}

	fmt.Fprintf(os.Stderr,
		bgColor+color.Bold+" %s%s "+color.Reset+" %s\n",
		upperName, code, msg.Text,
	)
	fmt.Fprintf(os.Stderr,
		color.Faint+"  at "+color.Reset+"%s:%d:%d:\n",
		msg.Location.File, msg.Location.Line, msg.Location.Column,
	)
	fmt.Fprint(os.Stderr,
		color.Faint+"  |"+color.Reset+"\n",
	)

	var firstNonWhitespaceChar, numLeadingTabs int
	for i, c := range msg.Location.LineText {
		if c == '\t' {
			numLeadingTabs += 1
		}
		if !unicode.IsSpace(c) {
			firstNonWhitespaceChar = i
			break
		}
	}
	extraSpacesBecauseOfTabs := numLeadingTabs * (4 - 1) // assuming we render tabs as 4 spaces
	lineNormalized := strings.Repeat(" ", firstNonWhitespaceChar+extraSpacesBecauseOfTabs) + msg.Location.LineText[firstNonWhitespaceChar:]

	fmt.Fprintf(os.Stderr,
		color.Faint+"  | "+color.Reset+"%s\n",
		lineNormalized,
	)
	fmt.Fprintf(os.Stderr,
		color.Faint+"  | "+color.Reset+"%s"+fgColor+"^ %s occurs here"+color.Reset+"\n",
		strings.Repeat(" ", msg.Location.Column+extraSpacesBecauseOfTabs), lowerName,
	)
	fmt.Fprint(os.Stderr,
		color.Faint+"  |"+color.Reset+"\n",
	)

	for _, note := range msg.Notes {
		noteSpacing := ""
		noteText := note.Text
		if note.Location != nil {
			noteSpacing = strings.Repeat(" ", note.Location.Column+extraSpacesBecauseOfTabs)
			noteText = "^ " + noteText
		}
		fmt.Fprintf(os.Stderr,
			color.Faint+"  | "+color.Reset+"%s"+color.Blue+"%s"+color.Reset+"\n",
			noteSpacing, noteText,
		)
		fmt.Fprint(os.Stderr,
			color.Faint+"  |"+color.Reset+"\n",
		)
	}
}
