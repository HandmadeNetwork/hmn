package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(DropAuthGroups{})
}

type DropAuthGroups struct{}

func (m DropAuthGroups) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 8, 3, 2, 14, 18, 0, time.UTC))
}

func (m DropAuthGroups) Name() string {
	return "DropAuthGroups"
}

func (m DropAuthGroups) Description() string {
	return "Drop the auth groups table, and related tables"
}

func (m DropAuthGroups) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		DROP TABLE handmade_user_projects;
		CREATE TABLE handmade_user_projects (
			user_id INT NOT NULL REFERENCES auth_user (id) ON DELETE CASCADE,
			project_id INT NOT NULL REFERENCES handmade_project (id) ON DELETE CASCADE,
			PRIMARY KEY (user_id, project_id)
		);

		INSERT INTO handmade_user_projects (user_id, project_id)
			SELECT agroups.user_id, pg.project_id
			FROM
				handmade_project_groups AS pg
				JOIN auth_group AS ag ON ag.id = pg.group_id
				JOIN auth_user_groups AS agroups ON agroups.group_id = ag.id;
	`)
	if err != nil {
		return oops.New(err, "failed to recreate handmade_user_projects")
	}

	_, err = tx.Exec(ctx, `
		DROP TABLE auth_group_permissions;
		DROP TABLE auth_user_groups;
		DROP TABLE handmade_project_groups;

		DROP TABLE auth_group;
	`)
	if err != nil {
		return oops.New(err, "failed to drop group-related tables")
	}

	return nil
}

func (m DropAuthGroups) Down(ctx context.Context, tx pgx.Tx) error {
	panic("Implement me")
}
