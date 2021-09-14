package db

import (
	"fmt"
	"strings"
)

type QueryBuilder struct {
	sql  strings.Builder
	args []interface{}
}

/*
Adds the given SQL and arguments to the query. Any occurrences
of `$?` will be replaced with the correct argument number.

foo $? bar $? baz $?
foo ARG1 bar ARG2 baz $?
foo ARG1 bar ARG2 baz ARG3
*/
func (qb *QueryBuilder) Add(sql string, args ...interface{}) {
	numPlaceholders := strings.Count(sql, "$?")
	if numPlaceholders != len(args) {
		panic(fmt.Errorf("cannot add chunk to query; expected %d arguments but got %d", numPlaceholders, len(args)))
	}

	for _, arg := range args {
		sql = strings.Replace(sql, "$?", fmt.Sprintf("$%d", len(qb.args)+1), 1)
		qb.args = append(qb.args, arg)
	}

	qb.sql.WriteString(sql)
	qb.sql.WriteString("\n")
}

func (qb *QueryBuilder) String() string {
	return qb.sql.String()
}

func (qb *QueryBuilder) Args() []interface{} {
	return qb.args
}
