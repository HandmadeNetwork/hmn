package migrations

import (
	"context"
	"fmt"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v4"
)

func init() {
	registerMigration(DeleteInactiveUsers{})
}

type DeleteInactiveUsers struct{}

func (m DeleteInactiveUsers) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 8, 8, 13, 46, 55, 0, time.UTC))
}

func (m DeleteInactiveUsers) Name() string {
	return "DeleteInactiveUsers"
}

func (m DeleteInactiveUsers) Description() string {
	return "Delete inactive users and expired onetimetokens"
}

func (m DeleteInactiveUsers) Up(ctx context.Context, tx pgx.Tx) error {
	var err error
	res, err := tx.Exec(ctx,
		`
		DELETE FROM handmade_passwordresetrequest
		USING handmade_onetimetoken
		WHERE handmade_onetimetoken.expires < $1 AND handmade_passwordresetrequest.confirmation_token_id = handmade_onetimetoken.id;
		`,
		time.Now(),
	)
	if err != nil {
		return oops.New(err, "failed to delete password reset requests")
	}
	fmt.Printf("Deleted %v expired password reset requests.\n", res.RowsAffected())

	fmt.Printf("Deleting inactive users. This might take a minute.\n")
	res, err = tx.Exec(ctx,
		`
		DELETE FROM auth_user
		WHERE status = 1 AND date_joined < $1;
		`,
		time.Now().Add(-(time.Hour * 24 * 7)),
	)
	if err != nil {
		return oops.New(err, "failed to delete inactive users")
	}
	fmt.Printf("Deleted %v inactive users.\n", res.RowsAffected())

	_, err = tx.Exec(ctx, `
		ALTER TABLE handmade_onetimetoken
			DROP used;
		ALTER TABLE handmade_onetimetoken
			ADD owner_id INT REFERENCES auth_user(id) ON DELETE CASCADE;

		ALTER TABLE handmade_userpending
			DROP CONSTRAINT handma_activation_token_id_0b4a4b06_fk_handmade_onetimetoken_id;
	`)

	_, err = tx.Exec(ctx, `
		UPDATE handmade_userpending
			SET activation_token_id = NULL
			WHERE (SELECT count(*) AS ct FROM handmade_onetimetoken WHERE id = activation_token_id) = 0;
	`)

	res, err = tx.Exec(ctx,
		`
		DELETE FROM handmade_onetimetoken
		WHERE expires < $1
		`,
		time.Now(),
	)
	if err != nil {
		return oops.New(err, "failed to delete expired tokens")
	}
	fmt.Printf("Deleted %v expired tokens.\n", res.RowsAffected())

	fmt.Printf("Setting owner_id on onetimetoken\n")
	_, err = tx.Exec(ctx, `
		UPDATE handmade_onetimetoken
			SET owner_id = (SELECT id FROM auth_user WHERE username = (SELECT username FROM handmade_userpending WHERE activation_token_id = handmade_onetimetoken.id LIMIT 1))
			WHERE token_type = 1;
	`)
	if err != nil {
		return oops.New(err, "failed to set owner_id on onetimetoken")
	}
	_, err = tx.Exec(ctx, `
		UPDATE handmade_onetimetoken
			SET owner_id = (SELECT user_id FROM handmade_passwordresetrequest WHERE confirmation_token_id = handmade_onetimetoken.id)
			WHERE token_type = 2;
	`)
	if err != nil {
		return oops.New(err, "failed to set owner_id on onetimetoken")
	}

	fmt.Printf("Setting registration_ip on auth_user\n")
	_, err = tx.Exec(ctx, `
		UPDATE auth_user
			SET registration_ip = (SELECT ip FROM handmade_userpending WHERE handmade_userpending.username = auth_user.username LIMIT 1);
	`)
	if err != nil {
		return oops.New(err, "failed to set owner_id on onetimetoken")
	}

	return nil
}

func (m DeleteInactiveUsers) Down(ctx context.Context, tx pgx.Tx) error {
	panic("Implement me")
}
