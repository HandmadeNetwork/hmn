package db

import (
	"context"
	"fmt"

	"git.handmade.network/hmn/hmn/config"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

func NewConn() *pgx.Conn {
	conn, err := pgx.Connect(context.Background(), config.Config.Postgres.DSN())
	if err != nil {
		panic(fmt.Errorf("failed to create database connection: %w", err))
	}

	return conn
}

func NewConnPool(minConns, maxConns int32) *pgxpool.Pool {
	config, err := pgxpool.ParseConfig(config.Config.Postgres.DSN())

	config.MinConns = minConns
	config.MaxConns = maxConns

	conn, err := pgxpool.ConnectConfig(context.Background(), config)
	if err != nil {
		panic(fmt.Errorf("failed to create database connection pool: %w", err))
	}

	return conn
}
