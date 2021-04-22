package db

import (
	"context"
	"errors"
	"reflect"
	"strings"

	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/log/zerologadapter"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/rs/zerolog/log"
)

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

	for i, val := range vals {
		if val == nil {
			continue
		}

		field := followPathThroughStructs(result, it.fieldPaths[i])
		if field.Kind() == reflect.Ptr {
			field.Set(reflect.New(field.Type().Elem()))
			field = field.Elem()
		}

		switch field.Kind() {
		case reflect.Int:
			field.SetInt(reflect.ValueOf(val).Int())
		default:
			field.Set(reflect.ValueOf(val))
		}
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

func followPathThroughStructs(structVal reflect.Value, path []int) reflect.Value {
	if len(path) < 1 {
		panic("can't follow an empty path")
	}

	val := structVal
	for _, i := range path {
		if val.Kind() == reflect.Ptr && val.Type().Elem().Kind() == reflect.Struct {
			if val.IsNil() {
				val.Set(reflect.New(val.Type()))
			}
			val = val.Elem()
		}
		val = val.Field(i)
	}
	return val
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

func getColumnNamesAndPaths(destType reflect.Type, pathSoFar []int, prefix string) ([]string, [][]int, error) {
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
			if destType.Kind() == reflect.Ptr {
				fieldType = destType.Elem()
			}

			_, isRecognizedByPgtype := connInfo.DataTypeForValue(reflect.New(fieldType).Elem().Interface()) // if pgtype recognizes it, we don't need to dig in further for more `db` tags
			// NOTE: boy it would be nice if we didn't have to do reflect.New here, considering that pgtype is just doing reflection on the value anyway

			if fieldType.Kind() == reflect.Struct && !isRecognizedByPgtype {
				subCols, subPaths, err := getColumnNamesAndPaths(fieldType, path, columnName+".")
				if err != nil {
					return nil, nil, err
				}
				columnNames = append(columnNames, subCols...)
				fieldPaths = append(fieldPaths, subPaths...)
			} else {
				columnNames = append(columnNames, prefix+columnName)
				fieldPaths = append(fieldPaths, path)
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
