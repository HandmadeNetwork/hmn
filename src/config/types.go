package config

import (
	"fmt"

	"github.com/jackc/pgx/v4"
)

type Environment string

const (
	Live Environment = "live"
	Beta             = "beta"
	Dev              = "dev"
)

type HMNConfig struct {
	Env      Environment
	Addr     string
	BaseUrl  string
	Postgres PostgresConfig
}

type PostgresConfig struct {
	User     string
	Password string
	Hostname string
	Port     int
	DbName   string
	LogLevel pgx.LogLevel
}

func (info PostgresConfig) DSN() string {
	return fmt.Sprintf("user=%s password=%s host=%s port=%d dbname=%s", info.User, info.Password, info.Hostname, info.Port, info.DbName)
}
