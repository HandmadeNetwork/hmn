package db

import (
	"context"
	"errors"
	"reflect"
	"strings"

	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/log/zerologadapter"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/rs/zerolog/log"
)

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
	fieldIndices []int
	rows         pgx.Rows
}

func (it *StructQueryIterator) Next(dest interface{}) bool {
	hasNext := it.rows.Next()
	if !hasNext {
		return false
	}

	v := reflect.ValueOf(dest)
	if v.Kind() != reflect.Ptr {
		panic(oops.New(nil, "Next requires a pointer type; got %v", v.Kind()))
	}

	vals, err := it.rows.Values()
	if err != nil {
		panic(err)
	}

	for i, val := range vals {
		field := v.Elem().Field(it.fieldIndices[i])
		switch field.Kind() {
		case reflect.Int:
			field.SetInt(reflect.ValueOf(val).Int())
		case reflect.Ptr:
			// TODO: I'm pretty sure we don't handle nullable ints correctly lol. Maybe this needs to be a function somehow, and recurse onto itself?? Reflection + recursion sounds like a great idea
			if val != nil {
				field.Set(reflect.New(field.Type().Elem()))
				field.Elem().Set(reflect.ValueOf(val))
			}
		default:
			field.Set(reflect.ValueOf(val))
		}
	}

	return true
}

func (it *StructQueryIterator) Close() {
	it.rows.Close()
}

func QueryToStructs(ctx context.Context, conn *pgxpool.Pool, destType interface{}, query string, args ...interface{}) (StructQueryIterator, error) {
	var fieldIndices []int
	var columnNames []string

	t := reflect.TypeOf(destType)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return StructQueryIterator{}, oops.New(nil, "QueryToStructs requires a struct type or a pointer to a struct type")
	}

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if columnName := f.Tag.Get("db"); columnName != "" {
			fieldIndices = append(fieldIndices, i)
			columnNames = append(columnNames, columnName)
		}
	}

	columnNamesString := strings.Join(columnNames, ", ")
	query = strings.Replace(query, "$columns", columnNamesString, -1)

	rows, err := conn.Query(ctx, query, args...)
	if err != nil {
		return StructQueryIterator{}, err
	}

	return StructQueryIterator{
		fieldIndices: fieldIndices,
		rows:         rows,
	}, nil
}

var ErrNoMatchingRows = errors.New("no matching rows")

func QueryOneToStruct(ctx context.Context, conn *pgxpool.Pool, dest interface{}, query string, args ...interface{}) error {
	rows, err := QueryToStructs(ctx, conn, dest, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	hasRow := rows.Next(dest)
	if !hasRow {
		return ErrNoMatchingRows
	}

	return nil
}
