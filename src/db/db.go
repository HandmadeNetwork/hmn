package db

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/google/uuid"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/log/zerologadapter"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/rs/zerolog/log"
)

/*
Values of these kinds are ok to query even if they are not directly understood by pgtype.
This is common for custom types like:

	type ThreadType int
*/
var queryableKinds = []reflect.Kind{
	reflect.Int,
}

/*
Checks if we are able to handle a particular type in a database query. This applies only to
primitive types and not structs, since the database only returns individual primitive types
and it is our job to stitch them back together into structs later.
*/
func typeIsQueryable(t reflect.Type) bool {
	_, isRecognizedByPgtype := connInfo.DataTypeForValue(reflect.New(t).Elem().Interface()) // if pgtype recognizes it, we don't need to dig in further for more `db` tags
	// NOTE: boy it would be nice if we didn't have to do reflect.New here, considering that pgtype is just doing reflection on the value anyway

	if isRecognizedByPgtype {
		return true
	} else if t == reflect.TypeOf(uuid.UUID{}) {
		return true
	}

	// pgtype doesn't recognize it, but maybe it's a primitive type we can deal with
	k := t.Kind()
	for _, qk := range queryableKinds {
		if k == qk {
			return true
		}
	}

	return false
}

// This interface should match both a direct pgx connection or a pgx transaction.
type ConnOrTx interface {
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
	Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error)

	// Both raw database connections and transactions in pgx can begin/commit
	// transactions. For database connections it does the obvious thing; for
	// transactions it creates a "pseudo-nested transaction" but conceptually
	// works the same. See the documentation of pgx.Tx.Begin.
	Begin(ctx context.Context) (pgx.Tx, error)
}

var connInfo = pgtype.NewConnInfo()

func NewConn() *pgx.Conn {
	conn, err := pgx.Connect(context.Background(), config.Config.Postgres.DSN())
	if err != nil {
		panic(oops.New(err, "failed to connect to database"))
	}

	return conn
}

func NewConnPool(minConns, maxConns int32) *pgxpool.Pool {
	cfg, err := pgxpool.ParseConfig(config.Config.Postgres.DSN())

	cfg.MinConns = minConns
	cfg.MaxConns = maxConns
	cfg.ConnConfig.Logger = zerologadapter.NewLogger(log.Logger)
	cfg.ConnConfig.LogLevel = config.Config.Postgres.LogLevel

	conn, err := pgxpool.ConnectConfig(context.Background(), cfg)
	if err != nil {
		panic(oops.New(err, "failed to create database connection pool"))
	}

	return conn
}

type StructQueryIterator struct {
	fieldPaths [][]int
	rows       pgx.Rows
	destType   reflect.Type
	closed     chan struct{}
}

func (it *StructQueryIterator) Next() (interface{}, bool) {
	hasNext := it.rows.Next()
	if !hasNext {
		it.Close()
		return nil, false
	}

	result := reflect.New(it.destType)

	vals, err := it.rows.Values()
	if err != nil {
		panic(err)
	}

	// Better logging of panics in this confusing reflection process
	var currentField reflect.StructField
	var currentValue reflect.Value
	var currentIdx int
	defer func() {
		if r := recover(); r != nil {
			if currentValue.IsValid() {
				logging.Error().
					Int("index", currentIdx).
					Str("field name", currentField.Name).
					Stringer("field type", currentField.Type).
					Interface("value", currentValue.Interface()).
					Stringer("value type", currentValue.Type()).
					Msg("panic in iterator")
			}

			if currentField.Name != "" {
				panic(fmt.Errorf("panic while processing field '%s': %v", currentField.Name, r))
			} else {
				panic(r)
			}
		}
	}()

	for i, val := range vals {
		currentIdx = i
		if val == nil {
			continue
		}

		var field reflect.Value
		field, currentField = followPathThroughStructs(result, it.fieldPaths[i])
		if field.Kind() == reflect.Ptr {
			field.Set(reflect.New(field.Type().Elem()))
			field = field.Elem()
		}

		// Some actual values still come through as pointers (like net.IPNet). Dunno why.
		// Regardless, we know it's not nil, so we can get at the contents.
		valReflected := reflect.ValueOf(val)
		if valReflected.Kind() == reflect.Ptr {
			valReflected = valReflected.Elem()
		}
		currentValue = valReflected

		switch field.Kind() {
		case reflect.Int:
			field.SetInt(valReflected.Int())
		default:
			field.Set(valReflected)
		}

		currentField = reflect.StructField{}
		currentValue = reflect.Value{}
	}

	return result.Interface(), true
}

func (it *StructQueryIterator) Close() {
	it.rows.Close()
	select {
	case it.closed <- struct{}{}:
	default:
	}
}

func (it *StructQueryIterator) ToSlice() []interface{} {
	defer it.Close()
	var result []interface{}
	for {
		row, ok := it.Next()
		if !ok {
			err := it.rows.Err()
			if err != nil {
				panic(oops.New(err, "error while iterating through db results"))
			}
			break
		}
		result = append(result, row)
	}
	return result
}

func followPathThroughStructs(structPtrVal reflect.Value, path []int) (reflect.Value, reflect.StructField) {
	if len(path) < 1 {
		panic(oops.New(nil, "can't follow an empty path"))
	}

	if structPtrVal.Kind() != reflect.Ptr || structPtrVal.Elem().Kind() != reflect.Struct {
		panic(oops.New(nil, "structPtrVal must be a pointer to a struct; got value of type %s", structPtrVal.Type()))
	}

	// more informative panic recovery
	var field reflect.StructField
	defer func() {
		if r := recover(); r != nil {
			panic(oops.New(nil, "panic at field '%s': %v", field.Name, r))
		}
	}()

	val := structPtrVal
	for _, i := range path {
		if val.Kind() == reflect.Ptr && val.Type().Elem().Kind() == reflect.Struct {
			if val.IsNil() {
				val.Set(reflect.New(val.Type().Elem()))
			}
			val = val.Elem()
		}
		field = val.Type().Field(i)
		val = val.Field(i)
	}
	return val, field
}

func Query(ctx context.Context, conn ConnOrTx, destExample interface{}, query string, args ...interface{}) ([]interface{}, error) {
	it, err := QueryIterator(ctx, conn, destExample, query, args...)
	if err != nil {
		return nil, err
	} else {
		return it.ToSlice(), nil
	}
}

func QueryIterator(ctx context.Context, conn ConnOrTx, destExample interface{}, query string, args ...interface{}) (*StructQueryIterator, error) {
	destType := reflect.TypeOf(destExample)
	columnNames, fieldPaths, err := getColumnNamesAndPaths(destType, nil, "")
	if err != nil {
		return nil, oops.New(err, "failed to generate column names")
	}

	columnNamesString := strings.Join(columnNames, ", ")
	query = strings.Replace(query, "$columns", columnNamesString, -1)

	rows, err := conn.Query(ctx, query, args...)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			panic("query exceeded its deadline")
		}
		return nil, err
	}

	it := &StructQueryIterator{
		fieldPaths: fieldPaths,
		rows:       rows,
		destType:   destType,
		closed:     make(chan struct{}, 1),
	}

	// Ensure that iterators are closed if context is cancelled. Otherwise, iterators can hold
	// open connections even after a request is cancelled, causing the app to deadlock.
	go func() {
		done := ctx.Done()
		if done == nil {
			return
		}
		select {
		case <-done:
			it.Close()
		case <-it.closed:
		}
	}()

	return it, nil
}

func getColumnNamesAndPaths(destType reflect.Type, pathSoFar []int, prefix string) (names []string, paths [][]int, err error) {
	var columnNames []string
	var fieldPaths [][]int

	if destType.Kind() == reflect.Ptr {
		destType = destType.Elem()
	}

	if destType.Kind() != reflect.Struct {
		return nil, nil, oops.New(nil, "can only get column names and paths from a struct, got type '%v' (at prefix '%v')", destType.Name(), prefix)
	}

	type AnonPrefix struct {
		Path   []int
		Prefix string
	}
	var anonPrefixes []AnonPrefix

	for _, field := range reflect.VisibleFields(destType) {
		path := append(pathSoFar, field.Index...)

		if columnName := field.Tag.Get("db"); columnName != "" {
			if field.Anonymous {
				anonPrefixes = append(anonPrefixes, AnonPrefix{Path: field.Index, Prefix: columnName})
				continue
			} else {
				for _, anonPrefix := range anonPrefixes {
					if len(field.Index) > len(anonPrefix.Path) {
						equal := true
						for i := range anonPrefix.Path {
							if anonPrefix.Path[i] != field.Index[i] {
								equal = false
								break
							}
						}
						if equal {
							columnName = anonPrefix.Prefix + "." + columnName
							break
						}
					}
				}
			}
			fieldType := field.Type
			if fieldType.Kind() == reflect.Ptr {
				fieldType = fieldType.Elem()
			}

			if typeIsQueryable(fieldType) {
				columnNames = append(columnNames, prefix+columnName)
				fieldPaths = append(fieldPaths, path)
			} else if fieldType.Kind() == reflect.Struct {
				subCols, subPaths, err := getColumnNamesAndPaths(fieldType, path, columnName+".")
				if err != nil {
					return nil, nil, err
				}
				columnNames = append(columnNames, subCols...)
				fieldPaths = append(fieldPaths, subPaths...)
			} else {
				return nil, nil, oops.New(nil, "field '%s' in type %s has invalid type '%s'", field.Name, destType, field.Type)
			}
		}
	}

	return columnNames, fieldPaths, nil
}

/*
A general error to be used when no results are found. This is the error returned
by QueryOne, and can generally be used by other database helpers that fetch a single
result but find nothing.
*/
var NotFound = errors.New("not found")

func QueryOne(ctx context.Context, conn ConnOrTx, destExample interface{}, query string, args ...interface{}) (interface{}, error) {
	rows, err := QueryIterator(ctx, conn, destExample, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result, hasRow := rows.Next()
	if !hasRow {
		return nil, NotFound
	}

	return result, nil
}

func QueryScalar(ctx context.Context, conn ConnOrTx, query string, args ...interface{}) (interface{}, error) {
	rows, err := conn.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			panic(err)
		}

		if len(vals) != 1 {
			return nil, oops.New(nil, "you must query exactly one field with QueryScalar, not %v", len(vals))
		}

		return vals[0], nil
	}

	return nil, NotFound
}

func QueryString(ctx context.Context, conn ConnOrTx, query string, args ...interface{}) (string, error) {
	result, err := QueryScalar(ctx, conn, query, args...)
	if err != nil {
		return "", err
	}

	switch r := result.(type) {
	case string:
		return r, nil
	default:
		return "", oops.New(nil, "QueryString got a non-string result: %v", result)
	}
}

func QueryInt(ctx context.Context, conn ConnOrTx, query string, args ...interface{}) (int, error) {
	result, err := QueryScalar(ctx, conn, query, args...)
	if err != nil {
		return 0, err
	}

	switch r := result.(type) {
	case int:
		return r, nil
	case int32:
		return int(r), nil
	case int64:
		return int(r), nil
	default:
		return 0, oops.New(nil, "QueryInt got a non-int result: %v", result)
	}
}

func QueryBool(ctx context.Context, conn ConnOrTx, query string, args ...interface{}) (bool, error) {
	result, err := QueryScalar(ctx, conn, query, args...)
	if err != nil {
		return false, err
	}

	switch r := result.(type) {
	case bool:
		return r, nil
	default:
		return false, oops.New(nil, "QueryBool got a non-bool result: %v", result)
	}
}
