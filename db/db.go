package db

import (
	"context"

	"git.handmade.network/hmn/hmn/config"
	"git.handmade.network/hmn/hmn/oops"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

func NewConn() *pgx.Conn {
	conn, err := pgx.Connect(context.Background(), config.Config.Postgres.DSN())
	if err != nil {
		panic(oops.New(err, "failed to connect to database"))
	}

	return conn
}

func NewConnPool(minConns, maxConns int32) *pgxpool.Pool {
	config, err := pgxpool.ParseConfig(config.Config.Postgres.DSN())

	config.MinConns = minConns
	config.MaxConns = maxConns

	conn, err := pgxpool.ConnectConfig(context.Background(), config)
	if err != nil {
		panic(oops.New(err, "failed to create database connection pool"))
	}

	return conn
}
