package db

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/utils"
	"github.com/google/uuid"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/log/zerologadapter"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/rs/zerolog/log"
)

/*
A general error to be used when no results are found. This is the error returned
by QueryOne, and can generally be used by other database helpers that fetch a single
result but find nothing.
*/
var NotFound = errors.New("not found")

// This interface should match both a direct pgx connection or a pgx transaction.
type ConnOrTx interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error)

	// Both raw database connections and transactions in pgx can begin/commit
	// transactions. For database connections it does the obvious thing; for
	// transactions it creates a "pseudo-nested transaction" but conceptually
	// works the same. See the documentation of pgx.Tx.Begin.
	Begin(ctx context.Context) (pgx.Tx, error)
}

var connInfo = pgtype.NewConnInfo()

// Creates a new connection to the HMN database.
// This connection is not safe for concurrent use.
func NewConn() *pgx.Conn {
	return NewConnWithConfig(config.PostgresConfig{})
}

func NewConnWithConfig(cfg config.PostgresConfig) *pgx.Conn {
	cfg = overrideDefaultConfig(cfg)

	pgcfg, err := pgx.ParseConfig(cfg.DSN())

	pgcfg.Logger = zerologadapter.NewLogger(log.Logger)
	pgcfg.LogLevel = cfg.LogLevel

	conn, err := pgx.ConnectConfig(context.Background(), pgcfg)
	if err != nil {
		panic(oops.New(err, "failed to connect to database"))
	}

	return conn
}

// Creates a connection pool for the HMN database.
// The resulting pool is safe for concurrent use.
func NewConnPool() *pgxpool.Pool {
	return NewConnPoolWithConfig(config.PostgresConfig{})
}

func NewConnPoolWithConfig(cfg config.PostgresConfig) *pgxpool.Pool {
	cfg = overrideDefaultConfig(cfg)

	pgcfg, err := pgxpool.ParseConfig(cfg.DSN())

	pgcfg.MinConns = cfg.MinConn
	pgcfg.MaxConns = cfg.MaxConn
	pgcfg.ConnConfig.Logger = zerologadapter.NewLogger(log.Logger)
	pgcfg.ConnConfig.LogLevel = cfg.LogLevel

	conn, err := pgxpool.ConnectConfig(context.Background(), pgcfg)
	if err != nil {
		panic(oops.New(err, "failed to create database connection pool"))
	}

	return conn
}

func overrideDefaultConfig(cfg config.PostgresConfig) config.PostgresConfig {
	return config.PostgresConfig{
		User:     utils.OrDefault(cfg.User, config.Config.Postgres.User),
		Password: utils.OrDefault(cfg.Password, config.Config.Postgres.Password),
		Hostname: utils.OrDefault(cfg.Hostname, config.Config.Postgres.Hostname),
		Port:     utils.OrDefault(cfg.Port, config.Config.Postgres.Port),
		DbName:   utils.OrDefault(cfg.DbName, config.Config.Postgres.DbName),
		LogLevel: utils.OrDefault(cfg.LogLevel, config.Config.Postgres.LogLevel),
		MinConn:  utils.OrDefault(cfg.MinConn, config.Config.Postgres.MinConn),
		MaxConn:  utils.OrDefault(cfg.MaxConn, config.Config.Postgres.MaxConn),
	}
}

/*
Performs a SQL query and returns a slice of all the result rows. The query is just plain SQL, but make sure to read the package documentation for details. You must explicitly provide the type argument - this is how it knows what Go type to map the results to, and it cannot be inferred.

Any SQL query may be performed, including INSERT and UPDATE - as long as it returns a result set, you can use this. If the query does not return a result set, or you simply do not care about the result set, call Exec directly on your pgx connection.

This function always returns pointers to the values. This is convenient for structs, but for other types, you may wish to use QueryScalar.
*/
func Query[T any](
	ctx context.Context,
	conn ConnOrTx,
	query string,
	args ...any,
) ([]*T, error) {
	it, err := QueryIterator[T](ctx, conn, query, args...)
	if err != nil {
		return nil, err
	} else {
		return it.ToSlice(), nil
	}
}

/*
Identical to Query, but panics if there was an error.
*/
func MustQuery[T any](
	ctx context.Context,
	conn ConnOrTx,
	query string,
	args ...any,
) []*T {
	result, err := Query[T](ctx, conn, query, args...)
	if err != nil {
		panic(err)
	}
	return result
}

/*
Identical to Query, but returns only the first result row. If there are no
rows in the result set, returns NotFound.
*/
func QueryOne[T any](
	ctx context.Context,
	conn ConnOrTx,
	query string,
	args ...any,
) (*T, error) {
	rows, err := QueryIterator[T](ctx, conn, query, args...)
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

/*
Identical to QueryOne, but panics if there was an error.
*/
func MustQueryOne[T any](
	ctx context.Context,
	conn ConnOrTx,
	query string,
	args ...any,
) *T {
	result, err := QueryOne[T](ctx, conn, query, args...)
	if err != nil {
		panic(err)
	}
	return result
}

/*
Identical to Query, but returns concrete values instead of pointers. More convenient
for primitive types.
*/
func QueryScalar[T any](
	ctx context.Context,
	conn ConnOrTx,
	query string,
	args ...any,
) ([]T, error) {
	rows, err := QueryIterator[T](ctx, conn, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []T
	for {
		val, hasRow := rows.Next()
		if !hasRow {
			break
		}
		result = append(result, *val)
	}

	return result, nil
}

/*
Identical to QueryScalar, but panics if there was an error.
*/
func MustQueryScalar[T any](
	ctx context.Context,
	conn ConnOrTx,
	query string,
	args ...any,
) []T {
	result, err := QueryScalar[T](ctx, conn, query, args...)
	if err != nil {
		panic(err)
	}
	return result
}

/*
Identical to QueryScalar, but returns only the first result value. If there are
no rows in the result set, returns NotFound.
*/
func QueryOneScalar[T any](
	ctx context.Context,
	conn ConnOrTx,
	query string,
	args ...any,
) (T, error) {
	rows, err := QueryIterator[T](ctx, conn, query, args...)
	if err != nil {
		var zero T
		return zero, err
	}
	defer rows.Close()

	result, hasRow := rows.Next()
	if !hasRow {
		var zero T
		return zero, NotFound
	}

	return *result, nil
}

/*
Identical to QueryOneScalar, but panics if there was an error.
*/
func MustQueryOneScalar[T any](
	ctx context.Context,
	conn ConnOrTx,
	query string,
	args ...any,
) T {
	result, err := QueryOneScalar[T](ctx, conn, query, args...)
	if err != nil {
		panic(err)
	}
	return result
}

/*
Identical to Query, but returns the ResultIterator instead of automatically converting the results to a slice. The iterator must be closed after use.
*/
func QueryIterator[T any](
	ctx context.Context,
	conn ConnOrTx,
	query string,
	args ...any,
) (*Iterator[T], error) {
	var destExample T
	destType := reflect.TypeOf(destExample)

	compiled := compileQuery(query, destType)

	rows, err := conn.Query(ctx, compiled.query, args...)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			panic("query exceeded its deadline")
		}
		return nil, err
	}

	it := &Iterator[T]{
		fieldPaths:       compiled.fieldPaths,
		rows:             rows,
		destType:         compiled.destType,
		destTypeIsScalar: typeIsQueryable(compiled.destType),
		closed:           make(chan struct{}, 1),
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

/*
Identical to QueryIterator, but panics if there was an error.
*/
func MustQueryIterator[T any](
	ctx context.Context,
	conn ConnOrTx,
	query string,
	args ...any,
) *Iterator[T] {
	result, err := QueryIterator[T](ctx, conn, query, args...)
	if err != nil {
		panic(err)
	}
	return result
}

// TODO: QueryFunc?

type compiledQuery struct {
	query      string
	destType   reflect.Type
	fieldPaths []fieldPath
}

var reColumnsPlaceholder = regexp.MustCompile(`\$columns({(.*?)})?`)

func compileQuery(query string, destType reflect.Type) compiledQuery {
	columnsMatch := reColumnsPlaceholder.FindStringSubmatch(query)
	hasColumnsPlaceholder := columnsMatch != nil

	if hasColumnsPlaceholder {
		// The presence of the $columns placeholder means that the destination type
		// must be a struct, and we will plonk that struct's fields into the query.

		if destType.Kind() != reflect.Struct {
			panic("$columns can only be used when querying into a struct")
		}

		var prefix []string
		prefixText := columnsMatch[2]
		if prefixText != "" {
			prefix = []string{prefixText}
		}

		columnNames, fieldPaths := getColumnNamesAndPaths(destType, nil, prefix)

		columns := make([]string, 0, len(columnNames))
		for _, strSlice := range columnNames {
			tableName := strings.Join(strSlice[0:len(strSlice)-1], "_")
			fullName := strSlice[len(strSlice)-1]
			if tableName != "" {
				fullName = tableName + "." + fullName
			}
			columns = append(columns, fullName)
		}

		columnNamesString := strings.Join(columns, ", ")
		query = reColumnsPlaceholder.ReplaceAllString(query, columnNamesString)

		return compiledQuery{
			query:      query,
			destType:   destType,
			fieldPaths: fieldPaths,
		}
	} else {
		return compiledQuery{
			query:    query,
			destType: destType,
		}
	}
}

func getColumnNamesAndPaths(destType reflect.Type, pathSoFar []int, prefix []string) (names []columnName, paths []fieldPath) {
	var columnNames []columnName
	var fieldPaths []fieldPath

	if destType.Kind() == reflect.Ptr {
		destType = destType.Elem()
	}

	if destType.Kind() != reflect.Struct {
		panic(fmt.Errorf("can only get column names and paths from a struct, got type '%v' (at prefix '%v')", destType.Name(), prefix))
	}

	type AnonPrefix struct {
		Path   []int
		Prefix string
	}
	var anonPrefixes []AnonPrefix

	for _, field := range reflect.VisibleFields(destType) {
		path := make([]int, len(pathSoFar))
		copy(path, pathSoFar)
		path = append(path, field.Index...)
		fieldColumnNames := prefix[:]

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
							fieldColumnNames = append(fieldColumnNames, anonPrefix.Prefix)
							break
						}
					}
				}
			}
			fieldType := field.Type
			if fieldType.Kind() == reflect.Ptr {
				fieldType = fieldType.Elem()
			}

			fieldColumnNames = append(fieldColumnNames, columnName)

			if typeIsQueryable(fieldType) {
				columnNames = append(columnNames, fieldColumnNames)
				fieldPaths = append(fieldPaths, path)
			} else if fieldType.Kind() == reflect.Struct {
				subCols, subPaths := getColumnNamesAndPaths(fieldType, path, fieldColumnNames)
				columnNames = append(columnNames, subCols...)
				fieldPaths = append(fieldPaths, subPaths...)
			} else {
				panic(fmt.Errorf("field '%s' in type %s has invalid type '%s'", field.Name, destType, field.Type))
			}
		}
	}

	return columnNames, fieldPaths
}

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

type columnName []string

// A path to a particular field in query's destination type. Each index in the slice
// corresponds to a field index for use with Field on a reflect.Type or reflect.Value.
type fieldPath []int

type Iterator[T any] struct {
	fieldPaths       []fieldPath
	rows             pgx.Rows
	destType         reflect.Type
	destTypeIsScalar bool // NOTE(ben): Make sure this gets set every time destType gets set, based on typeIsQueryable(destType). This is kinda fragile...but also contained to this file, so doesn't seem worth a lazy evaluation or a constructor function.
	closed           chan struct{}
}

func (it *Iterator[T]) Next() (*T, bool) {
	// TODO(ben): What happens if this panics? Does it leak resources? Do we need
	// to put a recover() here and close the rows?

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

	if it.destTypeIsScalar {
		// This type can be directly queried, meaning pgx recognizes it, it's
		// a simple scalar thing, and we can just take the easy way out.
		if len(vals) != 1 {
			panic(fmt.Errorf("tried to query a scalar value, but got %v values in the row", len(vals)))
		}
		setValueFromDB(result.Elem(), reflect.ValueOf(vals[0]))
		return result.Interface().(*T), true
	} else {
		var currentField reflect.StructField
		var currentValue reflect.Value
		var currentIdx int

		// Better logging of panics in this confusing reflection process
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

			setValueFromDB(field, valReflected)

			currentField = reflect.StructField{}
			currentValue = reflect.Value{}
		}

		return result.Interface().(*T), true
	}
}

func setValueFromDB(dest reflect.Value, value reflect.Value) {
	switch dest.Kind() {
	case reflect.Int:
		dest.SetInt(value.Int())
	default:
		dest.Set(value)
	}
}

func (it *Iterator[any]) Close() {
	it.rows.Close()
	select {
	case it.closed <- struct{}{}:
	default:
	}
}

/*
Pulls all the remaining values into a slice, and closes the iterator.
*/
func (it *Iterator[T]) ToSlice() []*T {
	defer it.Close()
	var result []*T
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
