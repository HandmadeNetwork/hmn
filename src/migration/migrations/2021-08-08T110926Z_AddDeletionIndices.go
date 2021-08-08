package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v4"
)

func init() {
	registerMigration(AddDeletionIndices{})
}

type AddDeletionIndices struct{}

func (m AddDeletionIndices) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 8, 8, 11, 9, 26, 0, time.UTC))
}

func (m AddDeletionIndices) Name() string {
	return "AddDeletionIndices"
}

func (m AddDeletionIndices) Description() string {
	return "Add indices to tables that depend on auth_user to allow for faster deletion of users"
}

func (m AddDeletionIndices) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `DROP TABLE auth_user_user_permissions;`)
	if err != nil {
		return oops.New(err, "failed to drop auth_user_user_permissions")
	}

	_, err = tx.Exec(ctx, `DROP TABLE django_admin_log;`)
	if err != nil {
		return oops.New(err, "failed to drop django_admin_log")
	}

	_, err = tx.Exec(ctx, `
		CREATE INDEX handmade_communicationchoice_userid ON handmade_communicationchoice (user_id);
		CREATE INDEX handmade_communicationsubcategory_userid ON handmade_communicationsubcategory (user_id);
		CREATE INDEX handmade_communicationsubthread_userid ON handmade_communicationsubthread (user_id);
		CREATE INDEX handmade_discord_hmnuserid ON handmade_discord (hmn_user_id);
		CREATE INDEX handmade_links_userid ON handmade_links (user_id);
		CREATE INDEX handmade_passwordresetrequest_userid ON handmade_passwordresetrequest (user_id);
		CREATE INDEX handmade_post_authorid ON handmade_post (author_id);
		CREATE INDEX handmade_postversion_editorid ON handmade_postversion (editor_id);
		CREATE INDEX handmade_thread_personalarticleuserid ON handmade_thread (personal_article_user_id);
		CREATE INDEX handmade_user_projects_userid ON handmade_user_projects (user_id);
	`)
	if err != nil {
		return oops.New(err, "failed to create user_id indices")
	}
	return nil
}

func (m AddDeletionIndices) Down(ctx context.Context, tx pgx.Tx) error {
	panic("Implement me")
}
