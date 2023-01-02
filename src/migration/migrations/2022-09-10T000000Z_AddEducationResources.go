package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(AddEducationResources{})
}

type AddEducationResources struct{}

func (m AddEducationResources) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2022, 9, 10, 0, 0, 0, 0, time.UTC))
}

func (m AddEducationResources) Name() string {
	return "AddEducationResources"
}

func (m AddEducationResources) Description() string {
	return "Adds the tables needed for the 2022 education initiative"
}

func (m AddEducationResources) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		CREATE TABLE education_article_version (
			id SERIAL NOT NULL PRIMARY KEY
		);

		CREATE TABLE education_article (
			id SERIAL NOT NULL PRIMARY KEY,
			title VARCHAR(255) NOT NULL,
			slug VARCHAR(255) NOT NULL UNIQUE,
			description TEXT NOT NULL,
			type INT NOT NULL,
			published BOOLEAN NOT NULL DEFAULT FALSE,
			current_version INT NOT NULL REFERENCES education_article_version (id) DEFERRABLE INITIALLY DEFERRED
		);

		ALTER TABLE education_article_version
			ADD article_id INT NOT NULL REFERENCES education_article (id) ON DELETE CASCADE,
			ADD date TIMESTAMP WITH TIME ZONE NOT NULL,
			ADD content_raw TEXT NOT NULL,
			ADD content_html TEXT NOT NULL,
			ADD editor_id INT REFERENCES hmn_user (id) ON DELETE SET NULL DEFERRABLE INITIALLY DEFERRED;
		`,
	)
	if err != nil {
		return oops.New(err, "failed to create education tables")
	}

	_, err = tx.Exec(ctx,
		`
		ALTER TABLE hmn_user
			DROP edit_library,
			ADD education_role INT NOT NULL DEFAULT 0;
		`,
	)
	if err != nil {
		return oops.New(err, "failed to update user stuff")
	}

	return nil
}

func (m AddEducationResources) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		DROP TABLE education_article CASCADE;
		DROP TABLE education_article_version CASCADE;
	`)
	if err != nil {
		return oops.New(err, "failed to delete education tables")
	}

	_, err = tx.Exec(ctx, `
		ALTER TABLE hmn_user
			DROP education_role,
			ADD edit_library BOOLEAN NOT NULL DEFAULT FALSE;
	`)
	if err != nil {
		return oops.New(err, "failed to delete education tables")
	}

	return nil
}
