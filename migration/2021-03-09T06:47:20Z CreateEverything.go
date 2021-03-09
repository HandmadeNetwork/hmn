package migration

import (
	"time"

	"github.com/jackc/pgx/v4"
)

func init() {
	registerMigration(CreateEverything{})
}

type CreateEverything struct{}

func (m CreateEverything) Date() time.Time {
	return time.Date(2021, 3, 9, 6, 53, 0, 0, time.UTC)
}

func (m CreateEverything) Up(conn *pgx.Conn) {

}

func (m CreateEverything) Down(conn *pgx.Conn) {

}
