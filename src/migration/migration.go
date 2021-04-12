package migration

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/migration/migrations"
	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/website"
	"github.com/jackc/pgx/v4"
	"github.com/spf13/cobra"
)

var listMigrations bool

func init() {
	migrateCommand := &cobra.Command{
		Use:   "migrate [target migration id]",
		Short: "Run database migrations",
		Run: func(cmd *cobra.Command, args []string) {
			if listMigrations {
				ListMigrations()
				return
			}

			targetVersion := time.Time{}
			if len(args) > 0 {
				var err error
				targetVersion, err = time.Parse(time.RFC3339, args[0])
				if err != nil {
					fmt.Printf("ERROR: bad version string: %v", err)
					os.Exit(1)
				}
			}
			Migrate(types.MigrationVersion(targetVersion))
		},
	}
	migrateCommand.Flags().BoolVar(&listMigrations, "list", false, "List available migrations")

	makeMigrationCommand := &cobra.Command{
		Use:   "makemigration <name> <description>...",
		Short: "Create a new database migration file",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 2 {
				fmt.Printf("You must provide a name and a description.\n\n")
				cmd.Usage()
				os.Exit(1)
			}

			name := args[0]
			description := strings.Join(args[1:], " ")

			MakeMigration(name, description)
		},
	}

	seedFromFileCommand := &cobra.Command{
		Use:   "seedfile <filename> <after migration id>",
		Short: "Resets the db, runs migrations up to and including <after migration id>, and runs the seed file.",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 2 {
				fmt.Printf("You must provide a seed file and migration id.\n\n")
				cmd.Usage()
				os.Exit(1)
			}

			seedFile := args[0]

			afterMigration, err := time.Parse(time.RFC3339, args[1])
			if err != nil {
				fmt.Printf("ERROR: bad version string: %v", err)
				os.Exit(1)
			}

			SeedFromFile(seedFile, types.MigrationVersion(afterMigration))
		},
	}

	website.WebsiteCommand.AddCommand(migrateCommand)
	website.WebsiteCommand.AddCommand(makeMigrationCommand)
	website.WebsiteCommand.AddCommand(seedFromFileCommand)
}

func getSortedMigrationVersions() []types.MigrationVersion {
	var allVersions []types.MigrationVersion
	for migrationTime, _ := range migrations.All {
		allVersions = append(allVersions, migrationTime)
	}
	sort.Slice(allVersions, func(i, j int) bool {
		return allVersions[i].Before(allVersions[j])
	})

	return allVersions
}

func getCurrentVersion(conn *pgx.Conn) (types.MigrationVersion, error) {
	var currentVersion time.Time
	row := conn.QueryRow(context.Background(), "SELECT version FROM hmn_migration")
	err := row.Scan(&currentVersion)
	if err != nil {
		return types.MigrationVersion{}, err
	}
	currentVersion = currentVersion.UTC()

	return types.MigrationVersion(currentVersion), nil
}

func ListMigrations() {
	conn := db.NewConn()
	defer conn.Close(context.Background())

	currentVersion, _ := getCurrentVersion(conn)
	for _, version := range getSortedMigrationVersions() {
		migration := migrations.All[version]
		indicator := "  "
		if version.Equal(currentVersion) {
			indicator = "✔ "
		}
		fmt.Printf("%s%v (%s: %s)\n", indicator, version, migration.Name(), migration.Description())
	}
}

func Migrate(targetVersion types.MigrationVersion) {
	conn := db.NewConn()
	defer conn.Close(context.Background())

	// create migration table
	_, err := conn.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS hmn_migration (
			version		TIMESTAMP WITH TIME ZONE
		)
	`)
	if err != nil {
		panic(fmt.Errorf("failed to create migration table: %w", err))
	}

	// ensure there is a row
	row := conn.QueryRow(context.Background(), "SELECT COUNT(*) FROM hmn_migration")
	var numRows int
	err = row.Scan(&numRows)
	if err != nil {
		panic(err)
	}
	if numRows < 1 {
		_, err := conn.Exec(context.Background(), "INSERT INTO hmn_migration (version) VALUES ($1)", time.Time{})
		if err != nil {
			panic(fmt.Errorf("failed to insert initial migration row: %w", err))
		}
	}

	// run migrations
	currentVersion, err := getCurrentVersion(conn)
	if err != nil {
		panic(fmt.Errorf("failed to get current version: %w", err))
	}
	if currentVersion.IsZero() {
		fmt.Println("This is the first time you have run database migrations.")
	} else {
		fmt.Printf("Current version: %s\n", currentVersion.String())
	}

	allVersions := getSortedMigrationVersions()
	if targetVersion.IsZero() {
		targetVersion = allVersions[len(allVersions)-1]
	}

	currentIndex := -1
	targetIndex := -1
	for i, version := range allVersions {
		if currentVersion.Equal(version) {
			currentIndex = i
		}
		if targetVersion.Equal(version) {
			targetIndex = i
		}
	}

	if targetIndex < 0 {
		fmt.Printf("ERROR: Could not find migration with version %v\n", targetVersion)
		return
	}

	if currentIndex < targetIndex {
		// roll forward
		for i := currentIndex + 1; i <= targetIndex; i++ {
			version := allVersions[i]
			fmt.Printf("Applying migration %v\n", version)
			migration := migrations.All[version]

			tx, err := conn.Begin(context.Background())
			if err != nil {
				panic(fmt.Errorf("failed to start transaction: %w", err))
			}
			defer tx.Rollback(context.Background())

			err = migration.Up(tx)
			if err != nil {
				fmt.Printf("MIGRATION FAILED for migration %v.\n", version)
				fmt.Printf("Error: %v\n", err)
				return
			}

			_, err = tx.Exec(context.Background(), "UPDATE hmn_migration SET version = $1", version)
			if err != nil {
				panic(fmt.Errorf("failed to update version in migrations table: %w", err))
			}

			err = tx.Commit(context.Background())
			if err != nil {
				panic(fmt.Errorf("failed to commit transaction: %w", err))
			}
		}
	} else if currentIndex > targetIndex {
		// roll back
		for i := currentIndex; i > targetIndex; i-- {
			version := allVersions[i]
			previousVersion := types.MigrationVersion{}
			if i > 0 {
				previousVersion = allVersions[i-1]
			}

			tx, err := conn.Begin(context.Background())
			if err != nil {
				panic(fmt.Errorf("failed to start transaction: %w", err))
			}
			defer tx.Rollback(context.Background())

			fmt.Printf("Rolling back migration %v\n", version)
			migration := migrations.All[version]
			err = migration.Down(tx)
			if err != nil {
				fmt.Printf("MIGRATION FAILED for migration %v.\n", version)
				fmt.Printf("Error: %v\n", err)
				return
			}

			_, err = tx.Exec(context.Background(), "UPDATE hmn_migration SET version = $1", previousVersion)
			if err != nil {
				panic(fmt.Errorf("failed to update version in migrations table: %w", err))
			}

			err = tx.Commit(context.Background())
			if err != nil {
				panic(fmt.Errorf("failed to commit transaction: %w", err))
			}
		}
	} else {
		fmt.Println("Already migrated; nothing to do.")
	}
}

//go:embed migrationTemplate.txt
var migrationTemplate string

func MakeMigration(name, description string) {
	result := migrationTemplate
	result = strings.ReplaceAll(result, "%NAME%", name)
	result = strings.ReplaceAll(result, "%DESCRIPTION%", fmt.Sprintf("%#v", description))

	now := time.Now().UTC()
	nowConstructor := fmt.Sprintf("time.Date(%d, %d, %d, %d, %d, %d, 0, time.UTC)", now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second())
	result = strings.ReplaceAll(result, "%DATE%", nowConstructor)

	safeVersion := strings.ReplaceAll(types.MigrationVersion(now).String(), ":", "")
	filename := fmt.Sprintf("%v_%v.go", safeVersion, name)
	path := filepath.Join("src", "migration", "migrations", filename)

	err := os.WriteFile(path, []byte(result), 0644)
	if err != nil {
		panic(fmt.Errorf("failed to write migration file: %w", err))
	}

	fmt.Println("Successfully created migration file:")
	fmt.Println(path)
}

// Drops all tables
func ClearDatabase() {
	conn := db.NewConn()
	defer conn.Close(context.Background())

	dbName := config.Config.Postgres.DbName
	rows, err := conn.Query(context.Background(), "SELECT tablename FROM pg_tables WHERE tableowner = $1", dbName)
	if err != nil {
		panic(fmt.Errorf("couldn't fetch the list of tables owned by %s: %w", dbName, err))
	}

	tablesToDrop := []string{}

	for rows.Next() {
		var tableName string
		err = rows.Scan(&tableName)
		if err != nil {
			panic(fmt.Errorf("failed to fetch row from pg_tables: %w", err))
		}
		tablesToDrop = append(tablesToDrop, tableName)
	}
	rows.Close()

	for _, tableName := range tablesToDrop {
		conn.Exec(context.Background(), "DROP TABLE $1", tableName)
	}
}

// Applies a cloned db to the local db.
// Drops the db and runs migrations from scratch.
// Applies the seed after the migration specified in `afterMigration`, then runs the rest of the migrations.
func SeedFromFile(seedFile string, afterMigration types.MigrationVersion) {
	file, err := os.Open(seedFile)
	if err != nil {
		panic(fmt.Errorf("couldn't open seed file %s: %w", seedFile, err))
	}
	file.Close()

	migration := migrations.All[afterMigration]

	if migration == nil {
		panic(fmt.Errorf("could not find migration: %s", afterMigration))
	}

	fmt.Println("Clearing database...")
	ClearDatabase()

	fmt.Println("Running migrations...")
	Migrate(afterMigration)

	fmt.Println("Executing seed...")
	cmd := exec.Command("psql",
		"--single-transaction",
		"--dbname",
		config.Config.Postgres.DbName,
		"--host",
		config.Config.Postgres.Hostname,
		"--username",
		config.Config.Postgres.User,
		"--password",
		"-f",
		seedFile,
	)
	fmt.Println("Running command:", cmd)
	if err = cmd.Run(); err != nil {
		exitError, isExit := err.(*exec.ExitError)
		if isExit {
			panic(fmt.Errorf("failed to execute seed: %w\n%s", err, string(exitError.Stderr)))
		} else {
			panic(fmt.Errorf("failed to execute seed: %w", err))
		}
	}

	fmt.Println("Done! You may want to migrate forward from here.")
	ListMigrations()
}

// NOTE(asaf): This will be useful for open-sourcing the website, but is not yet necessary.
// Creates only what's necessary for a fresh deployment with no data
func BareMinimumSeed() {
}

// NOTE(asaf): This will be useful for open-sourcing the website, but is not yet necessary.
// Creates enough data for development
func SampleSeed() {
	// admin := CreateAdminUser("admin", "12345678")
	// user := CreateUser("regular_user", "12345678")
	// hmnProject := CreateProject("hmn", "Handmade Network")
	// Create category
	// Create thread
	// Create accepted user project
	// Create pending user project
	// Create showcase items
	// Create codelanguages
	// Create library and library resources
}
