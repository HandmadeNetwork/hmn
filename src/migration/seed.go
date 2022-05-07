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
	"git.handmade.network/hmn/hmn/src/hmndata"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
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
	admin := seedUser(ctx, tx, models.User{Username: "admin", Name: "Admin", Email: "admin@handmade.network", IsStaff: true})

	fmt.Println("Creating normal users (all with password \"password\")...")
	alice := seedUser(ctx, tx, models.User{Username: "alice", Name: "Alice"})
	bob := seedUser(ctx, tx, models.User{Username: "bob", Name: "Bob"})
	charlie := seedUser(ctx, tx, models.User{Username: "charlie", Name: "Charlie"})

	fmt.Println("Creating a spammer...")
	spammer := seedUser(ctx, tx, models.User{
		Username: "spam",
		Status:   models.UserStatusConfirmed,
		Name:     "Hot singletons in your local area",
		Bio:      "Howdy, everybody I go by Jarva seesharpe from Bangalore. In this way, assuming you need to partake in a shared global instance with me then, at that poi",
	})

	users := []*models.User{alice, bob, charlie, spammer}

	fmt.Println("Creating some forum threads...")
	for i := 0; i < 5; i++ {
		thread := seedThread(ctx, tx, models.Thread{})
		populateThread(ctx, tx, thread, users, rand.Intn(5)+1)
	}

	// spam-only thread
	{
		thread := seedThread(ctx, tx, models.Thread{})
		populateThread(ctx, tx, thread, []*models.User{spammer}, 1)
	}

	fmt.Println("Creating the news posts...")
	{
		for i := 0; i < 3; i++ {
			thread := seedThread(ctx, tx, models.Thread{Type: models.ThreadTypeProjectBlogPost})
			populateThread(ctx, tx, thread, []*models.User{admin, alice, bob, charlie}, rand.Intn(5)+1)
		}
	}

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

func seedThread(ctx context.Context, tx pgx.Tx, input models.Thread) *models.Thread {
	input.Type = utils.OrDefault(input.Type, models.ThreadTypeForumPost)

	var defaultSubforum *int
	if input.Type == models.ThreadTypeForumPost {
		id := 2
		defaultSubforum = &id
	}

	thread, err := db.QueryOne[models.Thread](ctx, tx,
		`
		INSERT INTO thread (
			title,
			type, sticky,
			project_id, subforum_id,
			first_id, last_id
		)
		VALUES (
			$1,
			$2, $3,
			$4, $5,
			$6, $7
		)
		RETURNING $columns
		`,
		utils.OrDefault(input.Title, lorem.Sentence(3, 8)),
		utils.OrDefault(input.Type, models.ThreadTypeForumPost), false,
		utils.OrDefault(input.ProjectID, models.HMNProjectID), utils.OrDefault(input.SubforumID, defaultSubforum),
		-1, -1,
	)
	if err != nil {
		panic(oops.New(err, "failed to create thread"))
	}

	return thread
}

func populateThread(ctx context.Context, tx pgx.Tx, thread *models.Thread, users []*models.User, numPosts int) {
	var lastPostId int
	for i := 0; i < numPosts; i++ {
		user := users[i%len(users)]

		var replyId *int
		if lastPostId != 0 {
			if rand.Intn(10) < 3 {
				replyId = &lastPostId
			}
		}

		hmndata.CreateNewPost(ctx, tx, thread.ProjectID, thread.ID, thread.Type, user.ID, replyId, lorem.Paragraph(1, 10), "192.168.2.1")
	}
}

func randomName() string {
	return "John Doe" // chosen by fair dice roll. guaranteed to be random.
}

func randomBool() bool {
	return rand.Intn(2) == 1
}
