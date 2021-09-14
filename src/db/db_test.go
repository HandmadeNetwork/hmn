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
	type Embedded struct {
		NoTag S
		Nested
	}

	names, paths, err := getColumnNamesAndPaths(reflect.TypeOf(Embedded{}), nil, "")
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
			{1, 0, 0}, {1, 0, 1}, {1, 0, 2}, {1, 0, 3}, {1, 0, 4}, {1, 0, 5},
			{1, 1, 0}, {1, 1, 1}, {1, 1, 2}, {1, 1, 3}, {1, 1, 4}, {1, 1, 5},
		}, paths)
		assert.True(t, len(names) == len(paths))
	}

	testStruct := Embedded{}
	for i, path := range paths {
		val, field := followPathThroughStructs(reflect.ValueOf(&testStruct), path)
		assert.True(t, val.IsValid())
		assert.True(t, strings.Contains(names[i], field.Name))
	}
}

func TestQueryBuilder(t *testing.T) {
	t.Run("happy time", func(t *testing.T) {
		var qb QueryBuilder
		qb.Add("SELECT stuff FROM thing WHERE foo = $? AND bar = $?", 3, "hello")
		qb.Add("AND (baz = $?)", true)

		assert.Equal(t, "SELECT stuff FROM thing WHERE foo = $1 AND bar = $2\nAND (baz = $3)\n", qb.String())
		assert.Equal(t, []interface{}{3, "hello", true}, qb.Args())
	})
	t.Run("too few arguments", func(t *testing.T) {
		var qb QueryBuilder
		assert.Panics(t, func() {
			qb.Add("HELLO $? $? $?", 1, 2)
		})
	})
	t.Run("too many arguments", func(t *testing.T) {
		var qb QueryBuilder
		assert.Panics(t, func() {
			qb.Add("HELLO $? $? $?", 1, 2, 3, 4)
		})
	})
}
