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
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/website"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/spf13/cobra"
)

var listMigrations bool

func init() {
	dbCommand := &cobra.Command{
		Use:   "db",
		Short: "Database-related commands",
	}

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

	seedCommand := &cobra.Command{
		Use:   "seed",
		Short: "Resets the db and populates it with sample data.",
		Run: func(cmd *cobra.Command, args []string) {
			ResetDB()
			SampleSeed()
		},
	}

	seedFromFileCommand := &cobra.Command{
		Use:   "seedfile <filename>",
		Short: "Resets the db and runs the seed file.",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 1 {
				fmt.Printf("You must provide a seed file.\n\n")
				cmd.Usage()
				os.Exit(1)
			}

			ResetDB()
			SeedFromFile(args[0])
		},
	}

	website.WebsiteCommand.AddCommand(dbCommand)
	dbCommand.AddCommand(migrateCommand)
	dbCommand.AddCommand(makeMigrationCommand)
	dbCommand.AddCommand(seedCommand)
	dbCommand.AddCommand(seedFromFileCommand)
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

func getCurrentVersion(ctx context.Context, conn *pgx.Conn) (types.MigrationVersion, error) {
	var currentVersion time.Time
	row := conn.QueryRow(ctx, "SELECT version FROM hmn_migration")
	err := row.Scan(&currentVersion)
	if err != nil {
		return types.MigrationVersion{}, err
	}
	currentVersion = currentVersion.UTC()

	return types.MigrationVersion(currentVersion), nil
}

func tryGetCurrentVersion(ctx context.Context) types.MigrationVersion {
	defer func() {
		recover()
	}()

	conn := db.NewConn()
	defer conn.Close(ctx)

	currentVersion, _ := getCurrentVersion(ctx, conn)

	return currentVersion
}

func ListMigrations() {
	ctx := context.Background()

	currentVersion := tryGetCurrentVersion(ctx)
	for _, version := range getSortedMigrationVersions() {
		migration := migrations.All[version]
		indicator := "  "
		if version.Equal(currentVersion) {
			indicator = "✔ "
		}
		fmt.Printf("%s%v (%s: %s)\n", indicator, version, migration.Name(), migration.Description())
	}
}

func LatestVersion() types.MigrationVersion {
	allVersions := getSortedMigrationVersions()
	return allVersions[len(allVersions)-1]
}

// Migrates either forward or backward to the selected migration version. You probably want to
// use LatestVersion to get the most recent migration.
func Migrate(targetVersion types.MigrationVersion) {
	ctx := context.Background() // In the future, this could actually do something cool.

	conn := db.NewConn()
	defer conn.Close(ctx)

	// create migration table
	_, err := conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS hmn_migration (
			version		TIMESTAMP WITH TIME ZONE
		)
	`)
	if err != nil {
		panic(fmt.Errorf("failed to create migration table: %w", err))
	}

	// ensure there is a row
	row := conn.QueryRow(ctx, "SELECT COUNT(*) FROM hmn_migration")
	var numRows int
	err = row.Scan(&numRows)
	if err != nil {
		panic(err)
	}
	if numRows < 1 {
		_, err := conn.Exec(ctx, "INSERT INTO hmn_migration (version) VALUES ($1)", time.Time{})
		if err != nil {
			panic(fmt.Errorf("failed to insert initial migration row: %w", err))
		}
	}

	// run migrations
	currentVersion, err := getCurrentVersion(ctx, conn)
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
		targetVersion = LatestVersion()
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
			migration := migrations.All[version]
			fmt.Printf("Applying migration %v (%v)\n", version, migration.Name())

			tx, err := conn.Begin(ctx)
			if err != nil {
				panic(fmt.Errorf("failed to start transaction: %w", err))
			}
			defer tx.Rollback(ctx)

			err = migration.Up(ctx, tx)
			if err != nil {
				fmt.Printf("MIGRATION FAILED for migration %v.\n", version)
				fmt.Printf("Error: %v\n", err)
				return
			}

			_, err = tx.Exec(ctx, "UPDATE hmn_migration SET version = $1", version)
			if err != nil {
				panic(fmt.Errorf("failed to update version in migrations table: %w", err))
			}

			err = tx.Commit(ctx)
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

			tx, err := conn.Begin(ctx)
			if err != nil {
				panic(fmt.Errorf("failed to start transaction: %w", err))
			}
			defer tx.Rollback(ctx)

			fmt.Printf("Rolling back migration %v\n", version)
			migration := migrations.All[version]
			err = migration.Down(ctx, tx)
			if err != nil {
				fmt.Printf("MIGRATION FAILED for migration %v.\n", version)
				fmt.Printf("Error: %v\n", err)
				return
			}

			_, err = tx.Exec(ctx, "UPDATE hmn_migration SET version = $1", previousVersion)
			if err != nil {
				panic(fmt.Errorf("failed to update version in migrations table: %w", err))
			}

			err = tx.Commit(ctx)
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

func ResetDB() {
	fmt.Println("Resetting database...")

	ctx := context.Background()
	// NOTE(asaf): We connect to db "template1", because we have to connect to something other than our own db in order to drop it.
	template1DSN := fmt.Sprintf("user=%s password=%s host=%s port=%d dbname=%s",
		config.Config.Postgres.User,
		config.Config.Postgres.Password,
		config.Config.Postgres.Hostname,
		config.Config.Postgres.Port,
		"template1", // NOTE(asaf): template1 must always exist in postgres, as it's the db that gets cloned when you create new DBs
	)
	// NOTE(asaf): We have to use the low-level API of pgconn, because the pgx Exec always wraps the query in a transaction.
	lowLevelConn, err := pgconn.Connect(ctx, template1DSN)
	if err != nil {
		panic(fmt.Errorf("failed to connect to db: %w", err))
	}
	defer lowLevelConn.Close(ctx)

	// Disconnect all other users
	{
		result := lowLevelConn.ExecParams(ctx, fmt.Sprintf(`
			SELECT pg_terminate_backend(pid)
			FROM pg_stat_activity
			WHERE datname = '%s' AND pid <> pg_backend_pid()
		`, config.Config.Postgres.DbName), nil, nil, nil, nil)
		_, err := result.Close()
		if err != nil {
			panic(fmt.Errorf("failed to disconnect other users: %w", err))
		}
	}

	// Drop the database
	{
		result := lowLevelConn.ExecParams(ctx, fmt.Sprintf("DROP DATABASE %s", config.Config.Postgres.DbName), nil, nil, nil, nil)
		_, err = result.Close()
		pgErr, isPgError := err.(*pgconn.PgError)
		if err != nil {
			if !(isPgError && pgErr.SQLState() == "3D000") { // NOTE(asaf): 3D000 means "Database does not exist"
				panic(fmt.Errorf("failed to drop db: %w", err))
			}
		}
	}

	// Create the database again
	{
		result := lowLevelConn.ExecParams(ctx, fmt.Sprintf("CREATE DATABASE %s", config.Config.Postgres.DbName), nil, nil, nil, nil)
		_, err = result.Close()
		if err != nil {
			panic(fmt.Errorf("failed to create db: %w", err))
		}
	}
}

// Applies a cloned db to the local db.
// Applies the seed after the migration specified in `afterMigration`.
// NOTE(asaf): The db role specified in the config must have the CREATEDB attribute! `ALTER ROLE hmn WITH CREATEDB;`
func SeedFromFile(seedFile string) {
	file, err := os.Open(seedFile)
	if err != nil {
		panic(fmt.Errorf("couldn't open seed file %s: %w", seedFile, err))
	}
	file.Close()

	fmt.Println("Executing seed...")
	cmd := exec.Command("pg_restore",
		"--single-transaction",
		"--dbname", config.Config.Postgres.DSN(),
		seedFile,
	)
	fmt.Println("Running command:", cmd)
	if output, err := cmd.CombinedOutput(); err != nil {
		fmt.Print(string(output))
		panic(fmt.Errorf("failed to execute seed: %w", err))
	}

	fmt.Println("Done! You may want to migrate forward from here.")
	ListMigrations()
}

// NOTE(asaf): This will be useful for open-sourcing the website, but is not yet necessary.
// Creates only what's necessary for a fresh deployment with no data
// TODO(opensource)
func BareMinimumSeed() {
	Migrate(LatestVersion())

	ctx := context.Background()
	conn := db.NewConnPool(1, 1)
	defer conn.Close()

	tx, err := conn.Begin(ctx)
	if err != nil {
		panic(err)
	}
	defer tx.Rollback(ctx)

	// Create the HMN project
	_, err = tx.Exec(ctx,
		`
		INSERT INTO project (id, slug, name, blurb, description, personal, lifecycle, color_1, color_2, forum_enabled, blog_enabled, date_created)
		VALUES (1, 'hmn', 'Handmade Network', '', '', FALSE, $1, 'ab4c47', 'a5467d', TRUE, TRUE, '2017-01-01T00:00:00Z')
		`,
		models.ProjectLifecycleActive,
	)
	if err != nil {
		panic(err)
	}

	// Create the base forum
	_, err = tx.Exec(ctx, `
		INSERT INTO subforum (id, slug, name, parent_id, project_id)
		VALUES (2, '', 'Handmade Network', null, 1)
	`)
	if err != nil {
		panic(err)
	}

	// Associate the forum with the HMN project
	_, err = tx.Exec(ctx, `
		UPDATE project SET forum_id = 2 WHERE slug = 'hmn'
	`)
	if err != nil {
		panic(err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		panic(err)
	}
}

// NOTE(asaf): This will be useful for open-sourcing the website, but is not yet necessary.
// Creates enough data for development
// TODO(opensource)
func SampleSeed() {
	BareMinimumSeed()

	ctx := context.Background()
	conn := db.NewConnPool(1, 1)
	defer conn.Close()

	tx, err := conn.Begin(ctx)
	if err != nil {
		panic(err)
	}
	defer tx.Rollback(ctx)

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

	err = tx.Commit(ctx)
	if err != nil {
		panic(err)
	}
}
