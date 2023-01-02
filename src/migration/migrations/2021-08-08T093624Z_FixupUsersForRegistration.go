package migrations

import (
	"context"
	"fmt"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(FixupUsersForRegistration{})
}

type FixupUsersForRegistration struct{}

func (m FixupUsersForRegistration) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 8, 8, 9, 36, 24, 0, time.UTC))
}

func (m FixupUsersForRegistration) Name() string {
	return "FixupUsersForRegistration"
}

func (m FixupUsersForRegistration) Description() string {
	return "Remove PendingUser and add the necessary fields to users"
}

func (m FixupUsersForRegistration) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		ALTER TABLE auth_user
			ADD status INT NOT NULL DEFAULT 1;
		ALTER TABLE auth_user
			ADD registration_ip INET;
		ALTER TABLE auth_user
			ALTER COLUMN is_staff SET DEFAULT FALSE;
		ALTER TABLE auth_user
			ALTER COLUMN timezone SET DEFAULT 'UTC';
		ALTER TABLE auth_user
			DROP first_name;
		ALTER TABLE auth_user
			DROP last_name;
		ALTER TABLE auth_user
			DROP color_1;
		ALTER TABLE auth_user
			DROP color_2;
	`)
	if err != nil {
		return oops.New(err, "failed to modify auth_user")
	}

	fmt.Printf("Setting status on users.\n")
	// status = INACTIVE(1) when !is_active && last_login is null
	//			ACTIVE(2) when is_active
	//			BANNED(3) when !is_active && last_login is not null
	_, err = tx.Exec(ctx, `
		UPDATE auth_user
		SET status = CASE is_active WHEN TRUE THEN 2 ELSE (CASE WHEN last_login IS NULL THEN 1 ELSE 3 END) END;
	`)
	if err != nil {
		return oops.New(err, "failed to set user status")
	}

	_, err = tx.Exec(ctx, `
		ALTER TABLE auth_user
			DROP is_active;
	`)
	if err != nil {
		return oops.New(err, "failed to drop is_active")
	}

	return nil
}

func (m FixupUsersForRegistration) Down(ctx context.Context, tx pgx.Tx) error {
	panic("Implement me")
}
