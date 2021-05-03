package db

import (
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPaths(t *testing.T) {
	type CustomInt int
	type S struct {
		I   int        `db:"I"`
		PI  *int       `db:"PI"`
		CI  CustomInt  `db:"CI"`
		PCI *CustomInt `db:"PCI"`
		B   bool       `db:"B"`
		PB  *bool      `db:"PB"`

		NoTag int
	}
	type Nested struct {
		S  S  `db:"S"`
		PS *S `db:"PS"`

		NoTag S
	}

	names, paths, err := getColumnNamesAndPaths(reflect.TypeOf(Nested{}), nil, "")
	if assert.Nil(t, err) {
		assert.Equal(t, []string{
			"S.I", "S.PI",
			"S.CI", "S.PCI",
			"S.B", "S.PB",
			"PS.I", "PS.PI",
			"PS.CI", "PS.PCI",
			"PS.B", "PS.PB",
		}, names)
		assert.Equal(t, [][]int{
			{0, 0}, {0, 1}, {0, 2}, {0, 3}, {0, 4}, {0, 5},
			{1, 0}, {1, 1}, {1, 2}, {1, 3}, {1, 4}, {1, 5},
		}, paths)
		assert.True(t, len(names) == len(paths))
	}

	testStruct := Nested{}
	for i, path := range paths {
		val, field := followPathThroughStructs(reflect.ValueOf(&testStruct), path)
		assert.True(t, val.IsValid())
		assert.True(t, strings.Contains(names[i], field.Name))
	}
}
