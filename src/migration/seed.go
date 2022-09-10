package migration

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"time"

	"git.handmade.network/hmn/hmn/src/auth"
	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/hmndata"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/parsing"
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
func BareMinimumSeed() *models.Project {
	Migrate(LatestVersion())

	ctx := context.Background()
	conn := db.NewConnWithConfig(config.PostgresConfig{
		LogLevel: pgx.LogLevelWarn,
	})
	defer conn.Close(ctx)

	tx := utils.Must1(conn.Begin(ctx))
	defer tx.Rollback(ctx)

	fmt.Println("Creating HMN project...")
	hmn := seedProject(ctx, tx, seedHMN, nil)

	utils.Must(tx.Commit(ctx))

	return hmn
}

// Seeds the database with sample data for local dev.
func SampleSeed() {
	hmn := BareMinimumSeed()

	ctx := context.Background()
	conn := db.NewConnWithConfig(config.PostgresConfig{
		LogLevel: pgx.LogLevelWarn,
	})
	defer conn.Close(ctx)

	tx := utils.Must1(conn.Begin(ctx))
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

	fmt.Println("Creating starter projects...")
	hero := seedProject(ctx, tx, seedHandmadeHero, []*models.User{admin})
	fourcoder := seedProject(ctx, tx, seed4coder, []*models.User{bob})
	for i := 0; i < 5; i++ {
		name := fmt.Sprintf("%s %s", lorem.Word(1, 10), lorem.Word(1, 10))
		slug := strings.ReplaceAll(strings.ToLower(name), " ", "-")

		possibleOwners := []*models.User{alice, bob, charlie}
		var owners []*models.User
		for ownerIdx, owner := range possibleOwners {
			mask := (i % ((1 << len(possibleOwners)) - 1)) + 1
			if (1<<ownerIdx)&mask != 0 {
				owners = append(owners, owner)
			}
		}

		seedProject(ctx, tx, models.Project{
			Slug:        slug,
			Name:        name,
			Blurb:       lorem.Sentence(6, 16),
			Description: lorem.Paragraph(3, 5),

			Personal: true,
		}, owners)
	}
	// spam project!
	seedProject(ctx, tx, models.Project{
		Slug:        "spam",
		Name:        "Cheap abstraction enhancers",
		Blurb:       "Get higher than ever before...up the ladder of abstraction.",
		Description: "Tired of boring details like the actual problem assigned to you? The sky's the limit with these abstraction enhancers, guaranteed to sweep away all those pesky details so you can focus on what matters: \"architecture\".",

		Personal: true,
	}, []*models.User{spammer})

	fmt.Println("Creating some forum threads...")
	for i := 0; i < 5; i++ {
		for _, project := range []*models.Project{hmn, hero, fourcoder} {
			thread := seedThread(ctx, tx, project, models.Thread{})
			populateThread(ctx, tx, thread, users, rand.Intn(5)+1)
		}
	}
	// spam-only thread
	{
		thread := seedThread(ctx, tx, hmn, models.Thread{})
		populateThread(ctx, tx, thread, []*models.User{spammer}, 1)
	}

	fmt.Println("Creating news posts...")
	{
		// Main site news posts
		for i := 0; i < 3; i++ {
			thread := seedThread(ctx, tx, hmn, models.Thread{Type: models.ThreadTypeProjectBlogPost})
			populateThread(ctx, tx, thread, []*models.User{admin, alice, bob, charlie}, rand.Intn(5)+1)
		}

		// 4coder
		for i := 0; i < 5; i++ {
			thread := seedThread(ctx, tx, fourcoder, models.Thread{Type: models.ThreadTypeProjectBlogPost})
			populateThread(ctx, tx, thread, []*models.User{bob}, 1)
		}
	}

	// Finally, set sequence numbers to things that won't conflict
	utils.Must1(tx.Exec(ctx, "SELECT setval('project_id_seq', 100, true);"))

	utils.Must(tx.Commit(ctx))
}

func seedUser(ctx context.Context, conn db.ConnOrTx, input models.User) *models.User {
	user := db.MustQueryOne[models.User](ctx, conn,
		`
		INSERT INTO hmn_user (
			username, password, email,
			is_staff,
			status,
			name, bio, blurb, signature,
			darktheme,
			showemail,
			date_joined, registration_ip, avatar_asset_id
		)
		VALUES (
			$1, $2, $3,
			$4,
			$5,
			$6, $7, $8, $9,
			TRUE,
			$10,
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
	utils.Must(auth.SetPassword(ctx, conn, input.Username, "password"))

	return user
}

func seedThread(ctx context.Context, tx pgx.Tx, project *models.Project, input models.Thread) *models.Thread {
	input.Type = utils.OrDefault(input.Type, models.ThreadTypeForumPost)

	var defaultSubforum *int
	if input.Type == models.ThreadTypeForumPost {
		defaultSubforum = project.ForumID
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
		project.ID, utils.OrDefault(input.SubforumID, defaultSubforum),
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

var latestProjectId int

func seedProject(ctx context.Context, tx pgx.Tx, input models.Project, owners []*models.User) *models.Project {
	project := db.MustQueryOne[models.Project](ctx, tx,
		`
		INSERT INTO project (
			id,
			slug, name, blurb,
			description, descparsed,
			color_1, color_2,
			featured, personal, lifecycle, hidden,
			forum_enabled, blog_enabled,
			date_created
		)
		VALUES (
			$1,
			$2, $3, $4,
			$5, $6,
			$7, $8,
			$9, $10, $11, $12,
			$13, $14,
			$15
		)
		RETURNING $columns
		`,
		utils.OrDefault(input.ID, latestProjectId+1),
		input.Slug, input.Name, input.Blurb,
		input.Description, parsing.ParseMarkdown(input.Description, parsing.ForumRealMarkdown),
		input.Color1, input.Color2,
		input.Featured, input.Personal, utils.OrDefault(input.Lifecycle, models.ProjectLifecycleActive), input.Hidden,
		input.ForumEnabled, input.BlogEnabled,
		utils.OrDefault(input.DateCreated, time.Now()),
	)
	latestProjectId = utils.IntMax(latestProjectId, project.ID)

	// Create forum (even if unused)
	forum := db.MustQueryOne[models.Subforum](ctx, tx,
		`
		INSERT INTO subforum (
			slug, name,
			project_id
		)
		VALUES (
			$1, $2,
			$3
		)
		RETURNING $columns
		`,
		"", project.Name,
		project.ID,
	)

	// Associate forum with project
	utils.Must1(tx.Exec(ctx,
		`UPDATE project SET forum_id = $1 WHERE id = $2`,
		forum.ID, project.ID,
	))
	project.ForumID = &forum.ID

	// Add project owners
	for _, owner := range owners {
		utils.Must1(tx.Exec(ctx,
			`INSERT INTO user_project (user_id, project_id) VALUES ($1, $2)`,
			owner.ID, project.ID,
		))
	}

	return project
}

func randomName() string {
	return "John Doe" // chosen by fair dice roll. guaranteed to be random.
}

func randomBool() bool {
	return rand.Intn(2) == 1
}

var seedHMN = models.Project{
	ID:    models.HMNProjectID,
	Slug:  models.HMNProjectSlug,
	Name:  "Handmade Network",
	Blurb: "Changing the way software is written",
	Description: `
[project=hero]Originally inspired by Handmade Hero[/project], we're an offshoot of its community, hoping to change the way software is written. To this end we've circulated our [url=https://handmade.network/manifesto]manifesto[/url] and built this website, in the hopes of fostering this community. We invite others to host projects built with same goals in mind and build up or expand their community's reach in our little tree house and hope it proves vibrant soil for the exchange of ideas as well as code.
	`,

	Color1: "ab4c47", Color2: "a5467d",
	Hidden:       true,
	ForumEnabled: true, BlogEnabled: true,

	DateCreated: time.Date(2017, 1, 1, 0, 0, 0, 0, time.UTC),
}

var seedHandmadeHero = models.Project{
	Slug:  "hero",
	Name:  "Handmade Hero",
	Blurb: "An ongoing project to create a complete, professional-quality game accompanied by videos that explain every single line of its source code.",
	Description: `
Handmade Hero is an ongoing project by [Casey Muratori](http://mollyrocket.com/casey) to create a complete, professional-quality game accompanied by videos that explain every single line of its source code.  The series began on November 17th, 2014, and is estimated to run for at least 600 episodes.  Programming sessions are limited to one hour per weekday so it remains manageable for people who practice coding along with the series at home.

For more information, see the official website at https://handmadehero.org
	`,

	Color1: "19328a", Color2: "f1f0a2",
	Featured:     true,
	ForumEnabled: true, BlogEnabled: false,

	DateCreated: time.Date(2017, 1, 10, 0, 0, 0, 0, time.UTC),
}

var seed4coder = models.Project{
	Slug:  "4coder",
	Name:  "4coder",
	Blurb: "A programmable, cross platform, IDE template",
	Description: `
4coder preview video: https://www.youtube.com/watch?v=Nop5UW2kV3I

4coder differentiates from other editors by focusing on powerful C/C++ customization and extension, and ease of cross platform use.  This means that 4coder greatly reduces the cost of creating cross platform development tools such as debuggers, code intelligence systems.  It means that tools specialized to your particular needs can be programmed in C/C++ or any language that interfaces with C/C++, which is almost all of them.

In other words, 4coder is attempting to live in a space between an IDE and a power editor such as Emacs or Vim.

Want to try it out? [url=https://4coder.itch.io/4coder]Get your alpha build now[/url]!	
	`,

	Color1: "002107", Color2: "cccccc",
	ForumEnabled: true, BlogEnabled: true,

	DateCreated: time.Date(2017, 1, 10, 0, 0, 0, 0, time.UTC),
}
