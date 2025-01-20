package db

import (
	"context"
	"regexp"

	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/perf"
	"git.handmade.network/hmn/hmn/src/utils"
	zerologadapter "github.com/jackc/pgx-zerolog"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
)

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

// Creates a new connection to the HMN database.
// This connection is not safe for concurrent use.
func NewConn() *pgx.Conn {
	return NewConnWithConfig(config.PostgresConfig{})
}

func NewConnWithConfig(cfg config.PostgresConfig) *pgx.Conn {
	cfg = overrideDefaultConfig(cfg)

	pgcfg, err := pgx.ParseConfig(cfg.DSN())

	pgcfg.Tracer = multiTracer{
		&tracelog.TraceLog{
			Logger:   zerologadapter.NewLogger(*logging.GlobalLogger()),
			LogLevel: cfg.LogLevel,
		},
		requestPerfTracer{},
	}

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
	pgcfg.ConnConfig.Tracer = multiTracer{
		&tracelog.TraceLog{
			Logger:   zerologadapter.NewLogger(*logging.GlobalLogger()),
			LogLevel: cfg.LogLevel,
		},
		requestPerfTracer{},
	}

	conn, err := pgxpool.NewWithConfig(context.Background(), pgcfg)
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

type multiTracer []pgx.QueryTracer

var _ pgx.QueryTracer = multiTracer{}

func (mt multiTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	for _, t := range mt {
		ctx = t.TraceQueryStart(ctx, conn, data)
	}
	return ctx
}

func (mt multiTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	for _, t := range mt {
		t.TraceQueryEnd(ctx, conn, data)
	}
}

var reQueryName = regexp.MustCompile("---- (.*)\n")

func GetQueryName(sql string) (string, bool) {
	m := reQueryName.FindStringSubmatch(sql)
	if m != nil {
		return m[1], true
	}
	return "", false
}

const perfBlockContextKey = "__dbPerfBlock"

type requestPerfTracer struct{}

var _ pgx.QueryTracer = requestPerfTracer{}

func (pt requestPerfTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	p := perf.ExtractPerf(ctx)

	name := "Unknown query"
	if n, ok := GetQueryName(data.SQL); ok {
		name = n
	}
	b := p.StartBlock("SQL", name)
	return context.WithValue(ctx, perfBlockContextKey, b)
}

func (pt requestPerfTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	ctx.Value(perfBlockContextKey).(*perf.BlockHandle).End()
}
