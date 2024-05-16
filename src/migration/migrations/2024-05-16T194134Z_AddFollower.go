package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(AddFollower{})
}

type AddFollower struct{}

func (m AddFollower) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2024, 5, 16, 19, 41, 34, 0, time.UTC))
}

func (m AddFollower) Name() string {
	return "AddFollower"
}

func (m AddFollower) Description() string {
	return "Add follower table"
}

func (m AddFollower) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		CREATE TABLE follower (
			user_id int NOT NULL,
			following_user_id int REFERENCES hmn_user (id) ON DELETE CASCADE,
			following_project_id int REFERENCES project (id) ON DELETE CASCADE
		);

		CREATE INDEX follower_user_id ON follower(user_id);
		CREATE UNIQUE INDEX follower_following_user ON follower (user_id, following_user_id);
		CREATE UNIQUE INDEX follower_following_project ON follower (user_id, following_project_id);
		`,
	)
	return err
}

func (m AddFollower) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		DROP INDEX follower_following_user;
		DROP INDEX follower_following_project;
		DROP INDEX follower_user_id;
		DROP TABLE follower;
		`,
	)
	return err
}
