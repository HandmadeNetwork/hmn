package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(AddDismissedMembershipDiscordLinkBanner{})
}

type AddDismissedMembershipDiscordLinkBanner struct{}

func (m AddDismissedMembershipDiscordLinkBanner) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2026, 5, 24, 18, 0, 0, 0, time.UTC))
}

func (m AddDismissedMembershipDiscordLinkBanner) Name() string {
	return "AddDismissedMembershipDiscordLinkBanner"
}

func (m AddDismissedMembershipDiscordLinkBanner) Description() string {
	return "Track dismissal of the membership Discord link banner on hmn_user"
}

func (m AddDismissedMembershipDiscordLinkBanner) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		ALTER TABLE hmn_user
		ADD COLUMN dismissed_membership_discord_link_banner BOOLEAN NOT NULL DEFAULT false;
	`)
	return err
}

func (m AddDismissedMembershipDiscordLinkBanner) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		ALTER TABLE hmn_user
		DROP COLUMN dismissed_membership_discord_link_banner;
	`)
	return err
}
