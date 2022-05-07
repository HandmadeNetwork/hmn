package migration

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/exec"

	"git.handmade.network/hmn/hmn/src/auth"
	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/utils"
	lorem "github.com/HandmadeNetwork/golorem"
	"github.com/jackc/pgx/v4"
)

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

// Creates only what's necessary to get the site running. Not really very useful for
// local dev on its own; sample data makes things a lot better.
func BareMinimumSeed() {
	Migrate(LatestVersion())

	ctx := context.Background()
	conn := db.NewConnWithConfig(config.PostgresConfig{
		LogLevel: pgx.LogLevelWarn,
	})
	defer conn.Close(ctx)

	tx, err := conn.Begin(ctx)
	if err != nil {
		panic(err)
	}
	defer tx.Rollback(ctx)

	fmt.Println("Creating HMN project...")
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

	fmt.Println("Creating main forum...")
	_, err = tx.Exec(ctx, `
		INSERT INTO subforum (id, slug, name, parent_id, project_id)
		VALUES (2, '', 'Handmade Network', NULL, 1)
	`)
	if err != nil {
		panic(err)
	}
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

// Seeds the database with sample data for local dev.
func SampleSeed() {
	BareMinimumSeed()

	ctx := context.Background()
	conn := db.NewConnWithConfig(config.PostgresConfig{
		LogLevel: pgx.LogLevelWarn,
	})
	defer conn.Close(ctx)

	tx, err := conn.Begin(ctx)
	if err != nil {
		panic(err)
	}
	defer tx.Rollback(ctx)

	fmt.Println("Creating admin user (\"admin\"/\"password\")...")
	seedUser(ctx, conn, models.User{Username: "admin", Email: "admin@handmade.network", IsStaff: true})

	fmt.Println("Creating normal users (all with password \"password\")...")
	alice := seedUser(ctx, conn, models.User{Username: "alice", Name: "Alice"})
	bob := seedUser(ctx, conn, models.User{Username: "bob", Name: "Bob"})
	charlie := seedUser(ctx, conn, models.User{Username: "charlie", Name: "Charlie"})

	fmt.Println("Creating a spammer...")
	seedUser(ctx, conn, models.User{Username: "spam", Name: "Hot singletons in your local area", Status: models.UserStatusConfirmed})

	_ = []*models.User{alice, bob, charlie}

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

func seedUser(ctx context.Context, conn db.ConnOrTx, input models.User) *models.User {
	user, err := db.QueryOne[models.User](ctx, conn,
		`
		INSERT INTO hmn_user (
			username, password, email,
			is_staff,
			status,
			name, bio, blurb, signature,
			darktheme,
			showemail, edit_library,
			date_joined, registration_ip, avatar_asset_id
		)
		VALUES (
			$1, $2, $3,
			$4,
			$5,
			$6, $7, $8, $9,
			TRUE,
			$10, FALSE,
			'2017-01-01T00:00:00Z', '192.168.2.1', null
		)
		RETURNING $columns
		`,
		input.Username, "", utils.OrDefault(input.Email, fmt.Sprintf("%s@example.com", input.Username)),
		input.IsStaff,
		utils.OrDefault(input.Status, models.UserStatusApproved),
		utils.OrDefault(input.Name, randomName()), utils.OrDefault(input.Bio, lorem.Paragraph(0, 2)), utils.OrDefault(input.Blurb, lorem.Sentence(0, 14)), utils.OrDefault(input.Signature, lorem.Sentence(0, 16)),
		input.ShowEmail,
	)
	if err != nil {
		panic(err)
	}
	err = auth.SetPassword(ctx, conn, input.Username, "password")
	if err != nil {
		panic(err)
	}

	return user
}

func randomName() string {
	return "John Doe" // chosen by fair dice roll. guaranteed to be random.
}

func randomBool() bool {
	return rand.Intn(2) == 1
}
