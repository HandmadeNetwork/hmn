//go:build !js

package main

import (
	"fmt"
	"go/build"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"git.handmade.network/hmn/hmn/src/utils"
)

func main() {
	const publicDir = "../../../public"
	compile := exec.Command("go", "build", "-o", filepath.Join(publicDir, "parsing.wasm"))
	compile.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	run(compile)

	utils.Must(copy(
		fmt.Sprintf("%s/misc/wasm/wasm_exec.js", build.Default.GOROOT),
		filepath.Join(publicDir, "go_wasm_exec.js"),
	))
}

func run(cmd *exec.Cmd) {
	output, err := cmd.CombinedOutput()
	fmt.Print(string(output))
	if err != nil {
		fmt.Println(err)
		if exit, ok := err.(*exec.ExitError); ok {
			os.Exit(exit.ExitCode())
		}
	}
}

func copy(src string, dst string) error {
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()

	d, err := os.Create(dst)
	if err != nil {
		return err
	}

	if _, err := io.Copy(d, s); err != nil {
		d.Close()
		return err
	}

	return d.Close()
}
