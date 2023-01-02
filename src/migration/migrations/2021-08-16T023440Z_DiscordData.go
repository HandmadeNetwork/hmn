package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(DiscordData{})
}

type DiscordData struct{}

func (m DiscordData) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 8, 16, 2, 34, 40, 0, time.UTC))
}

func (m DiscordData) Name() string {
	return "DiscordData"
}

func (m DiscordData) Description() string {
	return "Clean up Discord data models"
}

func (m DiscordData) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		ALTER TABLE handmade_discord RENAME TO handmade_discorduser;
		ALTER TABLE handmade_discorduser
			ALTER username SET NOT NULL,
			ALTER discriminator SET NOT NULL,
			ALTER access_token SET NOT NULL,
			ALTER refresh_token SET NOT NULL,
			ALTER locale SET NOT NULL,
			ALTER userid SET NOT NULL,
			ALTER expiry SET NOT NULL;
	`)
	if err != nil {
		return oops.New(err, "failed to fix up discord table")
	}

	_, err = tx.Exec(ctx, `
		ALTER TABLE handmade_discordmessagecontent
			DROP CONSTRAINT handmade_discordmess_discord_id_1acc147f_fk_handmade_,
			DROP CONSTRAINT handmade_discordmess_message_id_4dfde67d_fk_handmade_,
			ADD FOREIGN KEY (discord_id) REFERENCES handmade_discorduser (id) ON DELETE CASCADE,
			ADD FOREIGN KEY (message_id) REFERENCES handmade_discordmessage (id) ON DELETE CASCADE;
		
		ALTER TABLE handmade_discordmessageattachment
			DROP CONSTRAINT handmade_discordmess_asset_id_c64a3c31_fk_handmade_,
			DROP CONSTRAINT handmade_discordmess_message_id_d39da9b3_fk_handmade_,
			ADD FOREIGN KEY (asset_id) REFERENCES handmade_asset (id) ON DELETE CASCADE,
			ADD FOREIGN KEY (message_id) REFERENCES handmade_discordmessage (id) ON DELETE CASCADE;
		
		ALTER TABLE handmade_discordmessageembed
			DROP CONSTRAINT handmade_discordmess_image_id_9b04bb5f_fk_handmade_,
			DROP CONSTRAINT handmade_discordmess_message_id_04f15ce6_fk_handmade_,
			DROP CONSTRAINT handmade_discordmess_video_id_1c41289f_fk_handmade_,
			ADD FOREIGN KEY (image_id) REFERENCES handmade_asset (id) ON DELETE SET NULL,
			ADD FOREIGN KEY (message_id) REFERENCES handmade_discordmessage (id) ON DELETE CASCADE,
			ADD FOREIGN KEY (video_id) REFERENCES handmade_asset (id) ON DELETE SET NULL;
	`)
	if err != nil {
		return oops.New(err, "failed to fix constraints")
	}

	return nil
}

func (m DiscordData) Down(ctx context.Context, tx pgx.Tx) error {
	panic("Implement me")
}
