package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v4"
)

func init() {
	registerMigration(RenameEverything{})
}

type RenameEverything struct{}

func (m RenameEverything) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2022, 5, 6, 22, 17, 26, 0, time.UTC))
}

func (m RenameEverything) Name() string {
	return "RenameEverything"
}

func (m RenameEverything) Description() string {
	return "Rename all the tables, and remove the ones we no longer need"
}

func (m RenameEverything) Up(ctx context.Context, tx pgx.Tx) error {
	// Drop unused tables
	_, err := tx.Exec(ctx, `
		DROP TABLE
			auth_permission,
			django_content_type,
			django_migrations,
			django_site,
			handmade_blacklistemail,
			handmade_blacklisthostname,
			handmade_codelanguage,
			handmade_communicationchoice,
			handmade_communicationchoicelist,
			handmade_communicationsubcategory,
			handmade_communicationsubthread,
			handmade_kunenathread,
			handmade_license,
			handmade_license_texts,
			handmade_project_languages,
			handmade_project_licenses
	`)
	if err != nil {
		return oops.New(err, "failed to drop unused tables")
	}

	// Rename everything!!
	_, err = tx.Exec(ctx, `
		ALTER TABLE auth_user RENAME TO hmn_user;
		ALTER TABLE discord_outgoingmessages RENAME TO discord_outgoing_message;
		ALTER TABLE handmade_asset RENAME TO asset;
		ALTER TABLE handmade_discordmessage RENAME TO discord_message;
		ALTER TABLE handmade_discordmessageattachment RENAME TO discord_message_attachment;
		ALTER TABLE handmade_discordmessagecontent RENAME TO discord_message_content;
		ALTER TABLE handmade_discordmessageembed RENAME TO discord_message_embed;
		ALTER TABLE handmade_discorduser RENAME TO discord_user;
		ALTER TABLE handmade_imagefile RENAME TO image_file;
		ALTER TABLE handmade_librarymediatype RENAME TO library_media_type;
		ALTER TABLE handmade_libraryresource RENAME TO library_resource;
		ALTER TABLE handmade_libraryresource_media_types RENAME TO library_resource_media_type;
		ALTER TABLE handmade_libraryresource_topics RENAME TO library_resource_topic;
		ALTER TABLE handmade_libraryresourcestar RENAME TO library_resource_star;
		ALTER TABLE handmade_librarytopic RENAME TO library_topic;
		ALTER TABLE handmade_links RENAME TO link;
		ALTER TABLE handmade_onetimetoken RENAME TO one_time_token;
		ALTER TABLE handmade_otherfile RENAME TO other_file;
		ALTER TABLE handmade_podcast RENAME TO podcast;
		ALTER TABLE handmade_podcastepisode RENAME TO podcast_episode;
		ALTER TABLE handmade_post RENAME TO post;
		ALTER TABLE handmade_post_asset_usage RENAME TO post_asset_usage;
		ALTER TABLE handmade_postversion RENAME TO post_version;
		ALTER TABLE handmade_project RENAME TO project;
		ALTER TABLE handmade_project_downloads RENAME TO project_download;
		ALTER TABLE handmade_project_screenshots RENAME TO project_screenshot;
		ALTER TABLE handmade_snippet RENAME TO snippet;
		ALTER TABLE handmade_subforum RENAME TO subforum;
		ALTER TABLE handmade_subforumlastreadinfo RENAME TO subforum_last_read_info;
		ALTER TABLE handmade_thread RENAME TO thread;
		ALTER TABLE handmade_threadlastreadinfo RENAME TO thread_last_read_info;
		ALTER TABLE handmade_user_projects RENAME TO user_project;
		ALTER TABLE sessions RENAME TO session;
		ALTER TABLE snippet_tags RENAME TO snippet_tag;
		ALTER TABLE tags RENAME TO tag;
		ALTER TABLE twitch_streams RENAME TO twitch_stream;

		ALTER SEQUENCE auth_user_id_seq RENAME TO hmn_user_id_seq;
		ALTER SEQUENCE discord_outgoingmessages_id_seq RENAME TO discord_outgoing_message_id_seq;
		ALTER SEQUENCE handmade_category_id_seq RENAME TO subforum_id_seq;
		ALTER SEQUENCE handmade_categorylastreadinfo_id_seq RENAME TO subforum_last_read_info_id_seq;
		ALTER SEQUENCE handmade_discord_id_seq RENAME TO discord_user_id_seq;
		ALTER SEQUENCE handmade_discordmessageembed_id_seq RENAME TO discord_message_embed_id_seq;
		ALTER SEQUENCE handmade_imagefile_id_seq RENAME TO image_file_id_seq;
		ALTER SEQUENCE handmade_librarymediatype_id_seq RENAME TO library_media_type_id_seq;
		ALTER SEQUENCE handmade_libraryresource_id_seq RENAME TO library_resource_id_seq;
		ALTER SEQUENCE handmade_libraryresource_media_types_id_seq RENAME TO library_resource_media_type_id_seq;
		ALTER SEQUENCE handmade_libraryresource_topics_id_seq RENAME TO library_resource_topic_id_seq;
		ALTER SEQUENCE handmade_libraryresourcestar_id_seq RENAME TO library_resource_star_id_seq;
		ALTER SEQUENCE handmade_librarytopic_id_seq RENAME TO library_topic_id_seq;
		ALTER SEQUENCE handmade_links_id_seq RENAME TO link_id_seq;
		ALTER SEQUENCE handmade_onetimetoken_id_seq RENAME TO one_time_token_id_seq;
		ALTER SEQUENCE handmade_otherfile_id_seq RENAME TO other_file_id_seq;
		ALTER SEQUENCE handmade_podcast_id_seq RENAME TO podcast_id_seq;
		ALTER SEQUENCE handmade_post_id_seq RENAME TO post_id_seq;
		ALTER SEQUENCE handmade_postversion_id_seq RENAME TO post_version_id_seq;
		ALTER SEQUENCE handmade_project_downloads_id_seq RENAME TO project_download_id_seq;
		ALTER SEQUENCE handmade_project_id_seq RENAME TO project_id_seq;
		ALTER SEQUENCE handmade_project_screenshots_id_seq RENAME TO project_screenshot_id_seq;
		ALTER SEQUENCE handmade_snippet_id_seq RENAME TO snippet_id_seq;
		ALTER SEQUENCE handmade_thread_id_seq RENAME TO thread_id_seq;
		ALTER SEQUENCE handmade_threadlastreadinfo_id_seq RENAME TO thread_last_read_info_id_seq;
		ALTER SEQUENCE tags_id_seq RENAME TO tag_id_seq;

		CREATE OR REPLACE FUNCTION thread_type_for_post(int) RETURNS int AS $$
			SELECT thread.type
			FROM
				public.post
				JOIN public.thread ON post.thread_id = thread.id
			WHERE post.id = $1
		$$ LANGUAGE SQL;

		CREATE OR REPLACE FUNCTION project_id_for_post(int) RETURNS int AS $$
			SELECT thread.project_id
			FROM
				public.post
				JOIN public.thread ON post.thread_id = thread.id
			WHERE post.id = $1
		$$ LANGUAGE SQL;
	`)
	if err != nil {
		return oops.New(err, "failed to rename tables")
	}

	return nil
}

func (m RenameEverything) Down(ctx context.Context, tx pgx.Tx) error {
	panic("Implement me")
}
