package test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"git.handmade.network/hmn/hmn/src/ansicolor"
	"github.com/stretchr/testify/assert"
)

/*
We have these tests in a separate package so that we can run the hmnurl package tests without
recursively invoking ourselves.
*/

// Test that all hmnurl functions starting with Build are covered by tests.
func TestRouteCoverage(t *testing.T) {
	tmp := t.TempDir()
	covFilePath := filepath.Join(tmp, "coverage.out")

	outputAndAssert(t, exec.Command("go", "test", "./..", "-coverprofile="+covFilePath), "failed to run hmnurl tests")
	coverageOutput := outputAndAssert(t, exec.Command("go", "tool", "cover", "-func="+covFilePath), "failed to run coverage tool")

	coverLineRe := regexp.MustCompile("(?P<name>\\w+)\\t+(?P<percent>[\\d.]+)%$")
	var uncoveredBuildFuncs []string
	for i, line := range bytes.Split(coverageOutput, []byte("\n")) {
		line := string(line)
		if line == "" || strings.HasPrefix(line, "total") {
			continue
		}

		matches := coverLineRe.FindStringSubmatch(line)
		if matches == nil {
			panic(fmt.Sprintf("line %d of coverage data could not be parsed (\"%s\")", i+1, line))
		}

		funcName := matches[coverLineRe.SubexpIndex("name")]
		coverPercentStr := matches[coverLineRe.SubexpIndex("percent")]

		if strings.HasPrefix(funcName, "Build") {
			coverPercent, err := strconv.ParseFloat(coverPercentStr, 64)
			if err != nil {
				panic(err)
			}

			if coverPercent == 0 {
				uncoveredBuildFuncs = append(uncoveredBuildFuncs, funcName)
			}
		}
	}

	if len(uncoveredBuildFuncs) > 0 {
		t.Logf("The following url Build functions were not covered by tests:\n")
		for _, funcName := range uncoveredBuildFuncs {
			t.Logf("%s\n", funcName)
		}
		t.FailNow()
	}
}

func outputAndAssert(t *testing.T, cmd *exec.Cmd, args ...interface{}) []byte {
	t.Helper()

	var stdout bytes.Buffer
	cmd.Stdout = io.MultiWriter(os.Stdout, &stdout)
	cmd.Stderr = os.Stderr

	fmt.Println(ansicolor.Gray + ansicolor.Italic + cmd.String() + ansicolor.Reset)

	fmt.Print(ansicolor.Gray)
	err := cmd.Run()
	fmt.Print(ansicolor.Reset)
	assert.Nil(t, err, args...)

	return stdout.Bytes()
}
