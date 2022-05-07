/*
This package contains lowish-level APIs for making database queries to our Postgres database. It streamlines the process of mapping query results to Go types, while allowing you to write arbitrary SQL queries.

The primary functions are Query and QueryIterator. See the package and function examples for detailed usage.

Query syntax

This package allows a few small extensions to SQL syntax to streamline the interaction between Go and Postgres.

Arguments can be provided using placeholders like $1, $2, etc. All arguments will be safely escaped and mapped from their Go type to the correct Postgres type. (This is a direct proxy to pgx.)

	projectIDs, err := db.Query[int](ctx, conn,
		`
		SELECT id
		FROM project
		WHERE
			slug = ANY($1)
			AND hidden = $2
		`,
		[]string{"4coder", "metadesk"},
		false,
	)

(This also demonstrates a useful tip: if you want to use a slice in your query, use Postgres arrays instead of IN.)

When querying individual fields, you can simply select the field like so:

	ids, err := db.Query[int](ctx, conn, `SELECT id FROM project`)

To query multiple columns at once, you may use a struct type with `db:"column_name"` tags, and the special $columns placeholder:

	type Project struct {
		ID          int       `db:"id"`
		Slug        string    `db:"slug"`
		DateCreated time.Time `db:"date_created"`
	}
	projects, err := db.Query[Project](ctx, conn, `SELECT $columns FROM ...`)
	// Resulting query:
	// SELECT id, slug, date_created FROM ...

Sometimes a table name prefix is required on each column to disambiguate between column names, especially when performing a JOIN. In those situations, you can include the prefix in the $columns placeholder like $columns{prefix}:

	type Project struct {
		ID          int       `db:"id"`
		Slug        string    `db:"slug"`
		DateCreated time.Time `db:"date_created"`
	}
	orphanedProjects, err := db.Query[Project](ctx, conn, `
		SELECT $columns{projects}
		FROM
			project AS projects
			LEFT JOIN user_project AS uproj
		WHERE
			uproj.user_id IS NULL
	`)
	// Resulting query:
	// SELECT projects.id, projects.slug, projects.date_created FROM ...
*/
package db
