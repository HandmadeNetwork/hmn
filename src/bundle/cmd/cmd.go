package cmd

import (
	"fmt"
	"go/build"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode"

	color "git.handmade.network/hmn/hmn/src/ansicolor"
	"git.handmade.network/hmn/hmn/src/bundle"
	"git.handmade.network/hmn/hmn/src/utils"
	"git.handmade.network/hmn/hmn/src/website"
	"github.com/evanw/esbuild/pkg/api"
	"github.com/spf13/cobra"
)

type WasmPackage struct {
	Package, Out string
}

var wasmPackages = []WasmPackage{
	// NOTE(ben): We currently lump everything into one file because Go wasm
	// modules are so intolerably huge. Best to only make the user download
	// 20 MB (!!!) once.
	{"./src/wasm", "gowasm.wasm"},
}

func init() {
	buildCommand := &cobra.Command{
		Use:   "bundle",
		Short: "Manually build the website CSS and JS. (Normally CSS and JS are built automatically during development.) Wasm can be built as a subcommand.",
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

	wasmCommand := &cobra.Command{
		Use:   "wasm",
		Short: "Build Wasm files for the frontend",
		Run: func(cmd *cobra.Command, args []string) {
			buildAllWasmFiles()
		},
	}
	buildCommand.AddCommand(wasmCommand)
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

func buildAllWasmFiles() {
	for _, wasmPackage := range wasmPackages {
		fmt.Fprintf(os.Stderr, "Building %s...\n", wasmPackage.Package)

		out := filepath.Join("public", wasmPackage.Out)
		compile := exec.Command("go", "build", "-o", out, wasmPackage.Package)
		compile.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
		compile.Stdout = os.Stdout
		compile.Stderr = os.Stderr
		if err := compile.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "WASM BUILD FAILED: %v\n", err)
			code := 1
			if exit, ok := err.(*exec.ExitError); ok {
				code = exit.ExitCode()
			}
			os.Exit(code)
		}
		// fmt.Fprintf(os.Stderr, "%s: size before opt = %v\n", out, utils.Must1(os.Stat(out)).Size())

		// opt := exec.Command("wasm-opt", "-Oz", "--all-features", "-g", out, "-o", out)
		// opt.Stdout = os.Stdout
		// opt.Stderr = os.Stderr
		// if err := opt.Run(); err != nil {
		// 	fmt.Fprintf(os.Stderr, "WASM-OPT FAILED: %v\n", err)
		// 	os.Exit(1)
		// }
		// fmt.Fprintf(os.Stderr, "%s: size after opt = %v\n", out, utils.Must1(os.Stat(out)).Size())
	}

	fmt.Fprintf(os.Stderr, "Copying wasm_exec.js...\n")
	utils.Must(copyFile(
		filepath.Join(build.Default.GOROOT, "lib/wasm/wasm_exec.js"),
		filepath.Join("public", "go_wasm_exec.js"),
	))
}

func copyFile(src, dst string) error {
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()

	d, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer d.Close()

	if _, err := io.Copy(d, s); err != nil {
		return err
	}

	return nil
}
