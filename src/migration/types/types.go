package types

import (
	"context"
	"time"

	"github.com/jackc/pgx/v4"
)

type Migration interface {
	Version() MigrationVersion
	Name() string
	Description() string
	Up(ctx context.Context, conn pgx.Tx) error
	Down(ctx context.Context, conn pgx.Tx) error
}

type MigrationVersion time.Time

func (v MigrationVersion) String() string {
	return time.Time(v).Format(time.RFC3339)
}

func (v MigrationVersion) Before(other MigrationVersion) bool {
	return time.Time(v).Before(time.Time(other))
}

func (v MigrationVersion) Equal(other MigrationVersion) bool {
	return time.Time(v).Equal(time.Time(other))
}

func (v MigrationVersion) IsZero() bool {
	return time.Time(v).IsZero()
}
