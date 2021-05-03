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
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/log/zerologadapter"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/rs/zerolog/log"
)

/*
Values of these kinds are ok to query even if they are not directly understood by pgtype.
This is common for custom types like:

	type CategoryKind int
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
}

func (it *StructQueryIterator) Next() (interface{}, bool) {
	hasNext := it.rows.Next()
	if !hasNext {
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
	defer func() {
		if r := recover(); r != nil {
			if currentValue.IsValid() {
				logging.Error().
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

func Query(ctx context.Context, conn *pgxpool.Pool, destExample interface{}, query string, args ...interface{}) (*StructQueryIterator, error) {
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

	return &StructQueryIterator{
		fieldPaths: fieldPaths,
		rows:       rows,
		destType:   destType,
	}, nil
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

	for i := 0; i < destType.NumField(); i++ {
		field := destType.Field(i)
		path := append(pathSoFar, i)

		if columnName := field.Tag.Get("db"); columnName != "" {
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

var ErrNoMatchingRows = errors.New("no matching rows")

func QueryOne(ctx context.Context, conn *pgxpool.Pool, destExample interface{}, query string, args ...interface{}) (interface{}, error) {
	rows, err := Query(ctx, conn, destExample, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result, hasRow := rows.Next()
	if !hasRow {
		return nil, ErrNoMatchingRows
	}

	return result, nil
}

func QueryScalar(ctx context.Context, conn *pgxpool.Pool, query string, args ...interface{}) (interface{}, error) {
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

	return nil, ErrNoMatchingRows
}

func QueryInt(ctx context.Context, conn *pgxpool.Pool, query string, args ...interface{}) (int, error) {
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
