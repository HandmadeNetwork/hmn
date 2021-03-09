package migration

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/db"
	"git.handmade.network/hmn/hmn/website"
	"github.com/jackc/pgx/v4"
	"github.com/spf13/cobra"
)

var migrations map[time.Time]Migration = make(map[time.Time]Migration)

type Migration interface {
	Date() time.Time
	Up(conn *pgx.Conn)
	Down(conn *pgx.Conn)
}

func registerMigration(m Migration) {
	migrations[m.Date()] = m
}

func init() {
	migrateCommand := &cobra.Command{
		Use:   "migrate",
		Short: "Run database migrations",
		Run: func(cmd *cobra.Command, args []string) {
			Migrate()
		},
	}

	website.WebsiteCommand.AddCommand(migrateCommand)
}

func Migrate() {
	conn := db.NewConn()
	defer conn.Close(context.Background())

	// check for existence of database??

	// check for migration data, create it if missing

	// run migrations
}
