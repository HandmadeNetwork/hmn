package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/migration/types"
	"github.com/jackc/pgx/v4"
)

func init() {
	registerMigration(Initial{})
}

type Initial struct{}

func (m Initial) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 3, 10, 5, 16, 21, 0, time.UTC))
}

func (m Initial) Name() string {
	return "Initial"
}

func (m Initial) Description() string {
	return "Creates all the tables from the old site"
}

func (m Initial) Up(tx pgx.Tx) error {
	_, err := tx.Exec(context.Background(), `
	--
	-- PostgreSQL database dump
	--
	
	-- Dumped from database version 9.6.18
	-- Dumped by pg_dump version 9.6.18
	
	--
	-- Name: auth_group; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.auth_group (
		id integer NOT NULL,
		name character varying(80) NOT NULL
	);
	
	
	ALTER TABLE public.auth_group OWNER TO hmn;
	
	--
	-- Name: auth_group_id_seq; Type: SEQUENCE; Schema: public; Owner: hmn
	--
	
	CREATE SEQUENCE public.auth_group_id_seq
		START WITH 1
		INCREMENT BY 1
		NO MINVALUE
		NO MAXVALUE
		CACHE 1;
	
	
	ALTER TABLE public.auth_group_id_seq OWNER TO hmn;
	
	--
	-- Name: auth_group_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: hmn
	--
	
	ALTER SEQUENCE public.auth_group_id_seq OWNED BY public.auth_group.id;
	
	
	--
	-- Name: auth_group_permissions; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.auth_group_permissions (
		id integer NOT NULL,
		group_id integer NOT NULL,
		permission_id integer NOT NULL
	);
	
	
	ALTER TABLE public.auth_group_permissions OWNER TO hmn;
	
	--
	-- Name: auth_group_permissions_id_seq; Type: SEQUENCE; Schema: public; Owner: hmn
	--
	
	CREATE SEQUENCE public.auth_group_permissions_id_seq
		START WITH 1
		INCREMENT BY 1
		NO MINVALUE
		NO MAXVALUE
		CACHE 1;
	
	
	ALTER TABLE public.auth_group_permissions_id_seq OWNER TO hmn;
	
	--
	-- Name: auth_group_permissions_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: hmn
	--
	
	ALTER SEQUENCE public.auth_group_permissions_id_seq OWNED BY public.auth_group_permissions.id;
	
	
	--
	-- Name: auth_permission; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.auth_permission (
		id integer NOT NULL,
		name character varying(255) NOT NULL,
		content_type_id integer NOT NULL,
		codename character varying(100) NOT NULL
	);
	
	
	ALTER TABLE public.auth_permission OWNER TO hmn;
	
	--
	-- Name: auth_permission_id_seq; Type: SEQUENCE; Schema: public; Owner: hmn
	--
	
	CREATE SEQUENCE public.auth_permission_id_seq
		START WITH 1
		INCREMENT BY 1
		NO MINVALUE
		NO MAXVALUE
		CACHE 1;
	
	
	ALTER TABLE public.auth_permission_id_seq OWNER TO hmn;
	
	--
	-- Name: auth_permission_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: hmn
	--
	
	ALTER SEQUENCE public.auth_permission_id_seq OWNED BY public.auth_permission.id;
	
	
	--
	-- Name: auth_user; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.auth_user (
		id integer NOT NULL,
		password character varying(128) NOT NULL,
		last_login timestamp with time zone,
		is_superuser boolean NOT NULL,
		username character varying(150) NOT NULL,
		first_name character varying(30) NOT NULL,
		last_name character varying(30) NOT NULL,
		email character varying(254) NOT NULL,
		is_staff boolean NOT NULL,
		is_active boolean NOT NULL,
		date_joined timestamp with time zone NOT NULL
	);
	
	
	ALTER TABLE public.auth_user OWNER TO hmn;
	
	--
	-- Name: auth_user_groups; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.auth_user_groups (
		id integer NOT NULL,
		user_id integer NOT NULL,
		group_id integer NOT NULL
	);
	
	
	ALTER TABLE public.auth_user_groups OWNER TO hmn;
	
	--
	-- Name: auth_user_groups_id_seq; Type: SEQUENCE; Schema: public; Owner: hmn
	--
	
	CREATE SEQUENCE public.auth_user_groups_id_seq
		START WITH 1
		INCREMENT BY 1
		NO MINVALUE
		NO MAXVALUE
		CACHE 1;
	
	
	ALTER TABLE public.auth_user_groups_id_seq OWNER TO hmn;
	
	--
	-- Name: auth_user_groups_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: hmn
	--
	
	ALTER SEQUENCE public.auth_user_groups_id_seq OWNED BY public.auth_user_groups.id;
	
	
	--
	-- Name: auth_user_id_seq; Type: SEQUENCE; Schema: public; Owner: hmn
	--
	
	CREATE SEQUENCE public.auth_user_id_seq
		START WITH 1
		INCREMENT BY 1
		NO MINVALUE
		NO MAXVALUE
		CACHE 1;
	
	
	ALTER TABLE public.auth_user_id_seq OWNER TO hmn;
	
	--
	-- Name: auth_user_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: hmn
	--
	
	ALTER SEQUENCE public.auth_user_id_seq OWNED BY public.auth_user.id;
	
	
	--
	-- Name: auth_user_user_permissions; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.auth_user_user_permissions (
		id integer NOT NULL,
		user_id integer NOT NULL,
		permission_id integer NOT NULL
	);
	
	
	ALTER TABLE public.auth_user_user_permissions OWNER TO hmn;
	
	--
	-- Name: auth_user_user_permissions_id_seq; Type: SEQUENCE; Schema: public; Owner: hmn
	--
	
	CREATE SEQUENCE public.auth_user_user_permissions_id_seq
		START WITH 1
		INCREMENT BY 1
		NO MINVALUE
		NO MAXVALUE
		CACHE 1;
	
	
	ALTER TABLE public.auth_user_user_permissions_id_seq OWNER TO hmn;
	
	--
	-- Name: auth_user_user_permissions_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: hmn
	--
	
	ALTER SEQUENCE public.auth_user_user_permissions_id_seq OWNED BY public.auth_user_user_permissions.id;
	
	
	--
	-- Name: django_admin_log; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.django_admin_log (
		id integer NOT NULL,
		action_time timestamp with time zone NOT NULL,
		object_id text,
		object_repr character varying(200) NOT NULL,
		action_flag smallint NOT NULL,
		change_message text NOT NULL,
		content_type_id integer,
		user_id integer NOT NULL,
		CONSTRAINT django_admin_log_action_flag_check CHECK ((action_flag >= 0))
	);
	
	
	ALTER TABLE public.django_admin_log OWNER TO hmn;
	
	--
	-- Name: django_admin_log_id_seq; Type: SEQUENCE; Schema: public; Owner: hmn
	--
	
	CREATE SEQUENCE public.django_admin_log_id_seq
		START WITH 1
		INCREMENT BY 1
		NO MINVALUE
		NO MAXVALUE
		CACHE 1;
	
	
	ALTER TABLE public.django_admin_log_id_seq OWNER TO hmn;
	
	--
	-- Name: django_admin_log_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: hmn
	--
	
	ALTER SEQUENCE public.django_admin_log_id_seq OWNED BY public.django_admin_log.id;
	
	
	--
	-- Name: django_content_type; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.django_content_type (
		id integer NOT NULL,
		app_label character varying(100) NOT NULL,
		model character varying(100) NOT NULL
	);
	
	
	ALTER TABLE public.django_content_type OWNER TO hmn;
	
	--
	-- Name: django_content_type_id_seq; Type: SEQUENCE; Schema: public; Owner: hmn
	--
	
	CREATE SEQUENCE public.django_content_type_id_seq
		START WITH 1
		INCREMENT BY 1
		NO MINVALUE
		NO MAXVALUE
		CACHE 1;
	
	
	ALTER TABLE public.django_content_type_id_seq OWNER TO hmn;
	
	--
	-- Name: django_content_type_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: hmn
	--
	
	ALTER SEQUENCE public.django_content_type_id_seq OWNED BY public.django_content_type.id;
	
	
	--
	-- Name: django_migrations; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.django_migrations (
		id integer NOT NULL,
		app character varying(255) NOT NULL,
		name character varying(255) NOT NULL,
		applied timestamp with time zone NOT NULL
	);
	
	
	ALTER TABLE public.django_migrations OWNER TO hmn;
	
	--
	-- Name: django_migrations_id_seq; Type: SEQUENCE; Schema: public; Owner: hmn
	--
	
	CREATE SEQUENCE public.django_migrations_id_seq
		START WITH 1
		INCREMENT BY 1
		NO MINVALUE
		NO MAXVALUE
		CACHE 1;
	
	
	ALTER TABLE public.django_migrations_id_seq OWNER TO hmn;
	
	--
	-- Name: django_migrations_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: hmn
	--
	
	ALTER SEQUENCE public.django_migrations_id_seq OWNED BY public.django_migrations.id;
	
	
	--
	-- Name: django_session; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.django_session (
		session_key character varying(40) NOT NULL,
		session_data text NOT NULL,
		expire_date timestamp with time zone NOT NULL
	);
	
	
	ALTER TABLE public.django_session OWNER TO hmn;
	
	--
	-- Name: django_site; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.django_site (
		id integer NOT NULL,
		domain character varying(100) NOT NULL,
		name character varying(50) NOT NULL
	);
	
	
	ALTER TABLE public.django_site OWNER TO hmn;
	
	--
	-- Name: django_site_id_seq; Type: SEQUENCE; Schema: public; Owner: hmn
	--
	
	CREATE SEQUENCE public.django_site_id_seq
		START WITH 1
		INCREMENT BY 1
		NO MINVALUE
		NO MAXVALUE
		CACHE 1;
	
	
	ALTER TABLE public.django_site_id_seq OWNER TO hmn;
	
	--
	-- Name: django_site_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: hmn
	--
	
	ALTER SEQUENCE public.django_site_id_seq OWNED BY public.django_site.id;
	
	
	--
	-- Name: handmade_asset; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.handmade_asset (
		id uuid NOT NULL,
		s3_key character varying(2000) NOT NULL,
		filename character varying(1000) NOT NULL,
		size integer NOT NULL,
		mime_type character varying(100) NOT NULL,
		sha1sum character varying(40) NOT NULL,
		width integer NOT NULL,
		height integer NOT NULL,
		uploader_id integer
	);
	
	
	ALTER TABLE public.handmade_asset OWNER TO hmn;
	
	--
	-- Name: handmade_blacklistemail; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.handmade_blacklistemail (
		email character varying(254) NOT NULL,
		source character varying(254) NOT NULL,
		verified integer NOT NULL
	);
	
	
	ALTER TABLE public.handmade_blacklistemail OWNER TO hmn;
	
	--
	-- Name: handmade_blacklisthostname; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.handmade_blacklisthostname (
		hostname character varying(254) NOT NULL,
		seen_ham integer NOT NULL,
		seen_spam integer NOT NULL
	);
	
	
	ALTER TABLE public.handmade_blacklisthostname OWNER TO hmn;
	
	--
	-- Name: handmade_category; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.handmade_category (
		id integer NOT NULL,
		slug character varying(30),
		name character varying(255),
		blurb character varying(140),
		kind integer NOT NULL,
		color_1 character varying(6) NOT NULL,
		color_2 character varying(6) NOT NULL,
		depth integer NOT NULL,
		parent_id integer,
		project_id integer
	);
	
	
	ALTER TABLE public.handmade_category OWNER TO hmn;
	
	--
	-- Name: handmade_category_id_seq; Type: SEQUENCE; Schema: public; Owner: hmn
	--
	
	CREATE SEQUENCE public.handmade_category_id_seq
		START WITH 1
		INCREMENT BY 1
		NO MINVALUE
		NO MAXVALUE
		CACHE 1;
	
	
	ALTER TABLE public.handmade_category_id_seq OWNER TO hmn;
	
	--
	-- Name: handmade_category_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: hmn
	--
	
	ALTER SEQUENCE public.handmade_category_id_seq OWNED BY public.handmade_category.id;
	
	
	--
	-- Name: handmade_categorylastreadinfo; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.handmade_categorylastreadinfo (
		id integer NOT NULL,
		lastread timestamp with time zone,
		member_id integer NOT NULL,
		category_id integer
	);
	
	
	ALTER TABLE public.handmade_categorylastreadinfo OWNER TO hmn;
	
	--
	-- Name: handmade_categorylastreadinfo_id_seq; Type: SEQUENCE; Schema: public; Owner: hmn
	--
	
	CREATE SEQUENCE public.handmade_categorylastreadinfo_id_seq
		START WITH 1
		INCREMENT BY 1
		NO MINVALUE
		NO MAXVALUE
		CACHE 1;
	
	
	ALTER TABLE public.handmade_categorylastreadinfo_id_seq OWNER TO hmn;
	
	--
	-- Name: handmade_categorylastreadinfo_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: hmn
	--
	
	ALTER SEQUENCE public.handmade_categorylastreadinfo_id_seq OWNED BY public.handmade_categorylastreadinfo.id;
	
	
	--
	-- Name: handmade_codelanguage; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.handmade_codelanguage (
		id integer NOT NULL,
		slug character varying(255) NOT NULL,
		name character varying(255) NOT NULL,
		description text,
		wikipedia character varying(255) NOT NULL
	);
	
	
	ALTER TABLE public.handmade_codelanguage OWNER TO hmn;
	
	--
	-- Name: handmade_codelanguage_id_seq; Type: SEQUENCE; Schema: public; Owner: hmn
	--
	
	CREATE SEQUENCE public.handmade_codelanguage_id_seq
		START WITH 1
		INCREMENT BY 1
		NO MINVALUE
		NO MAXVALUE
		CACHE 1;
	
	
	ALTER TABLE public.handmade_codelanguage_id_seq OWNER TO hmn;
	
	--
	-- Name: handmade_codelanguage_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: hmn
	--
	
	ALTER SEQUENCE public.handmade_codelanguage_id_seq OWNED BY public.handmade_codelanguage.id;
	
	
	--
	-- Name: handmade_communicationchoice; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.handmade_communicationchoice (
		id integer NOT NULL,
		choice integer NOT NULL,
		member_id integer NOT NULL,
		option_id integer NOT NULL
	);
	
	
	ALTER TABLE public.handmade_communicationchoice OWNER TO hmn;
	
	--
	-- Name: handmade_communicationchoice_id_seq; Type: SEQUENCE; Schema: public; Owner: hmn
	--
	
	CREATE SEQUENCE public.handmade_communicationchoice_id_seq
		START WITH 1
		INCREMENT BY 1
		NO MINVALUE
		NO MAXVALUE
		CACHE 1;
	
	
	ALTER TABLE public.handmade_communicationchoice_id_seq OWNER TO hmn;
	
	--
	-- Name: handmade_communicationchoice_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: hmn
	--
	
	ALTER SEQUENCE public.handmade_communicationchoice_id_seq OWNED BY public.handmade_communicationchoice.id;
	
	
	--
	-- Name: handmade_communicationchoicelist; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.handmade_communicationchoicelist (
		id integer NOT NULL,
		ordering integer NOT NULL,
		title character varying(255) NOT NULL,
		description text,
		project_id integer,
		key integer NOT NULL
	);
	
	
	ALTER TABLE public.handmade_communicationchoicelist OWNER TO hmn;
	
	--
	-- Name: handmade_communicationchoicelist_id_seq; Type: SEQUENCE; Schema: public; Owner: hmn
	--
	
	CREATE SEQUENCE public.handmade_communicationchoicelist_id_seq
		START WITH 1
		INCREMENT BY 1
		NO MINVALUE
		NO MAXVALUE
		CACHE 1;
	
	
	ALTER TABLE public.handmade_communicationchoicelist_id_seq OWNER TO hmn;
	
	--
	-- Name: handmade_communicationchoicelist_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: hmn
	--
	
	ALTER SEQUENCE public.handmade_communicationchoicelist_id_seq OWNED BY public.handmade_communicationchoicelist.id;
	
	
	--
	-- Name: handmade_communicationsubcategory; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.handmade_communicationsubcategory (
		id integer NOT NULL,
		choice integer NOT NULL,
		category_id integer NOT NULL,
		member_id integer NOT NULL
	);
	
	
	ALTER TABLE public.handmade_communicationsubcategory OWNER TO hmn;
	
	--
	-- Name: handmade_communicationsubcategory_id_seq; Type: SEQUENCE; Schema: public; Owner: hmn
	--
	
	CREATE SEQUENCE public.handmade_communicationsubcategory_id_seq
		START WITH 1
		INCREMENT BY 1
		NO MINVALUE
		NO MAXVALUE
		CACHE 1;
	
	
	ALTER TABLE public.handmade_communicationsubcategory_id_seq OWNER TO hmn;
	
	--
	-- Name: handmade_communicationsubcategory_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: hmn
	--
	
	ALTER SEQUENCE public.handmade_communicationsubcategory_id_seq OWNED BY public.handmade_communicationsubcategory.id;
	
	
	--
	-- Name: handmade_communicationsubthread; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.handmade_communicationsubthread (
		id integer NOT NULL,
		choice integer NOT NULL,
		member_id integer NOT NULL,
		thread_id integer NOT NULL
	);
	
	
	ALTER TABLE public.handmade_communicationsubthread OWNER TO hmn;
	
	--
	-- Name: handmade_communicationsubthread_id_seq; Type: SEQUENCE; Schema: public; Owner: hmn
	--
	
	CREATE SEQUENCE public.handmade_communicationsubthread_id_seq
		START WITH 1
		INCREMENT BY 1
		NO MINVALUE
		NO MAXVALUE
		CACHE 1;
	
	
	ALTER TABLE public.handmade_communicationsubthread_id_seq OWNER TO hmn;
	
	--
	-- Name: handmade_communicationsubthread_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: hmn
	--
	
	ALTER SEQUENCE public.handmade_communicationsubthread_id_seq OWNED BY public.handmade_communicationsubthread.id;
	
	
	--
	-- Name: handmade_discord; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.handmade_discord (
		id integer NOT NULL,
		username character varying(255),
		discriminator character varying(8),
		access_token character varying(255),
		refresh_token character varying(4096),
		member_id integer NOT NULL,
		avatar character varying(255),
		locale character varying(16),
		userid character varying(255),
		expiry timestamp with time zone
	);
	
	
	ALTER TABLE public.handmade_discord OWNER TO hmn;
	
	--
	-- Name: handmade_discord_id_seq; Type: SEQUENCE; Schema: public; Owner: hmn
	--
	
	CREATE SEQUENCE public.handmade_discord_id_seq
		START WITH 1
		INCREMENT BY 1
		NO MINVALUE
		NO MAXVALUE
		CACHE 1;
	
	
	ALTER TABLE public.handmade_discord_id_seq OWNER TO hmn;
	
	--
	-- Name: handmade_discord_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: hmn
	--
	
	ALTER SEQUENCE public.handmade_discord_id_seq OWNED BY public.handmade_discord.id;
	
	
	--
	-- Name: handmade_discordmessage; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.handmade_discordmessage (
		id character varying(255) NOT NULL,
		channel_id character varying(255) NOT NULL,
		guild_id character varying(255),
		url character varying(1000) NOT NULL,
		user_id character varying(255) NOT NULL,
		sent_at timestamp with time zone NOT NULL,
		snippet_created boolean NOT NULL
	);
	
	
	ALTER TABLE public.handmade_discordmessage OWNER TO hmn;
	
	--
	-- Name: handmade_discordmessageattachment; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.handmade_discordmessageattachment (
		id character varying(255) NOT NULL,
		asset_id uuid NOT NULL,
		message_id character varying(255) NOT NULL
	);
	
	
	ALTER TABLE public.handmade_discordmessageattachment OWNER TO hmn;
	
	--
	-- Name: handmade_discordmessagecontent; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.handmade_discordmessagecontent (
		message_id character varying(255) NOT NULL,
		last_content character varying(5000) NOT NULL,
		discord_id integer NOT NULL
	);
	
	
	ALTER TABLE public.handmade_discordmessagecontent OWNER TO hmn;
	
	--
	-- Name: handmade_discordmessageembed; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.handmade_discordmessageembed (
		id integer NOT NULL,
		title character varying(1000),
		description character varying(5000),
		url character varying(5000),
		image_id uuid,
		message_id character varying(255) NOT NULL,
		video_id uuid
	);
	
	
	ALTER TABLE public.handmade_discordmessageembed OWNER TO hmn;
	
	--
	-- Name: handmade_discordmessageembed_id_seq; Type: SEQUENCE; Schema: public; Owner: hmn
	--
	
	CREATE SEQUENCE public.handmade_discordmessageembed_id_seq
		START WITH 1
		INCREMENT BY 1
		NO MINVALUE
		NO MAXVALUE
		CACHE 1;
	
	
	ALTER TABLE public.handmade_discordmessageembed_id_seq OWNER TO hmn;
	
	--
	-- Name: handmade_discordmessageembed_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: hmn
	--
	
	ALTER SEQUENCE public.handmade_discordmessageembed_id_seq OWNED BY public.handmade_discordmessageembed.id;
	
	
	--
	-- Name: handmade_imagefile; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.handmade_imagefile (
		id integer NOT NULL,
		file character varying(255),
		size integer NOT NULL,
		sha1sum character varying(40) NOT NULL,
		protected boolean NOT NULL,
		height integer NOT NULL,
		width integer NOT NULL
	);
	
	
	ALTER TABLE public.handmade_imagefile OWNER TO hmn;
	
	--
	-- Name: handmade_imagefile_id_seq; Type: SEQUENCE; Schema: public; Owner: hmn
	--
	
	CREATE SEQUENCE public.handmade_imagefile_id_seq
		START WITH 1
		INCREMENT BY 1
		NO MINVALUE
		NO MAXVALUE
		CACHE 1;
	
	
	ALTER TABLE public.handmade_imagefile_id_seq OWNER TO hmn;
	
	--
	-- Name: handmade_imagefile_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: hmn
	--
	
	ALTER SEQUENCE public.handmade_imagefile_id_seq OWNED BY public.handmade_imagefile.id;
	
	
	--
	-- Name: handmade_kunenapost; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.handmade_kunenapost (
		kunenapost integer NOT NULL,
		post_id integer NOT NULL
	);
	
	
	ALTER TABLE public.handmade_kunenapost OWNER TO hmn;
	
	--
	-- Name: handmade_kunenathread; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.handmade_kunenathread (
		kunenathread integer NOT NULL,
		thread_id integer NOT NULL
	);
	
	
	ALTER TABLE public.handmade_kunenathread OWNER TO hmn;
	
	--
	-- Name: handmade_librarymediatype; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.handmade_librarymediatype (
		id integer NOT NULL,
		name character varying(100) NOT NULL
	);
	
	
	ALTER TABLE public.handmade_librarymediatype OWNER TO hmn;
	
	--
	-- Name: handmade_librarymediatype_id_seq; Type: SEQUENCE; Schema: public; Owner: hmn
	--
	
	CREATE SEQUENCE public.handmade_librarymediatype_id_seq
		START WITH 1
		INCREMENT BY 1
		NO MINVALUE
		NO MAXVALUE
		CACHE 1;
	
	
	ALTER TABLE public.handmade_librarymediatype_id_seq OWNER TO hmn;
	
	--
	-- Name: handmade_librarymediatype_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: hmn
	--
	
	ALTER SEQUENCE public.handmade_librarymediatype_id_seq OWNED BY public.handmade_librarymediatype.id;
	
	
	--
	-- Name: handmade_libraryresource; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.handmade_libraryresource (
		id integer NOT NULL,
		name character varying(255) NOT NULL,
		description character varying(2000) NOT NULL,
		url character varying(1000) NOT NULL,
		category_id integer NOT NULL,
		is_deleted boolean NOT NULL,
		project_id integer,
		content_type character varying(255) NOT NULL,
		size integer NOT NULL,
		prevents_embed boolean NOT NULL
	);
	
	
	ALTER TABLE public.handmade_libraryresource OWNER TO hmn;
	
	--
	-- Name: handmade_libraryresource_id_seq; Type: SEQUENCE; Schema: public; Owner: hmn
	--
	
	CREATE SEQUENCE public.handmade_libraryresource_id_seq
		START WITH 1
		INCREMENT BY 1
		NO MINVALUE
		NO MAXVALUE
		CACHE 1;
	
	
	ALTER TABLE public.handmade_libraryresource_id_seq OWNER TO hmn;
	
	--
	-- Name: handmade_libraryresource_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: hmn
	--
	
	ALTER SEQUENCE public.handmade_libraryresource_id_seq OWNED BY public.handmade_libraryresource.id;
	
	
	--
	-- Name: handmade_libraryresource_media_types; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.handmade_libraryresource_media_types (
		id integer NOT NULL,
		libraryresource_id integer NOT NULL,
		librarymediatype_id integer NOT NULL
	);
	
	
	ALTER TABLE public.handmade_libraryresource_media_types OWNER TO hmn;
	
	--
	-- Name: handmade_libraryresource_media_types_id_seq; Type: SEQUENCE; Schema: public; Owner: hmn
	--
	
	CREATE SEQUENCE public.handmade_libraryresource_media_types_id_seq
		START WITH 1
		INCREMENT BY 1
		NO MINVALUE
		NO MAXVALUE
		CACHE 1;
	
	
	ALTER TABLE public.handmade_libraryresource_media_types_id_seq OWNER TO hmn;
	
	--
	-- Name: handmade_libraryresource_media_types_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: hmn
	--
	
	ALTER SEQUENCE public.handmade_libraryresource_media_types_id_seq OWNED BY public.handmade_libraryresource_media_types.id;
	
	
	--
	-- Name: handmade_libraryresource_topics; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.handmade_libraryresource_topics (
		id integer NOT NULL,
		libraryresource_id integer NOT NULL,
		librarytopic_id integer NOT NULL
	);
	
	
	ALTER TABLE public.handmade_libraryresource_topics OWNER TO hmn;
	
	--
	-- Name: handmade_libraryresource_topics_id_seq; Type: SEQUENCE; Schema: public; Owner: hmn
	--
	
	CREATE SEQUENCE public.handmade_libraryresource_topics_id_seq
		START WITH 1
		INCREMENT BY 1
		NO MINVALUE
		NO MAXVALUE
		CACHE 1;
	
	
	ALTER TABLE public.handmade_libraryresource_topics_id_seq OWNER TO hmn;
	
	--
	-- Name: handmade_libraryresource_topics_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: hmn
	--
	
	ALTER SEQUENCE public.handmade_libraryresource_topics_id_seq OWNED BY public.handmade_libraryresource_topics.id;
	
	
	--
	-- Name: handmade_libraryresourcestar; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.handmade_libraryresourcestar (
		id integer NOT NULL,
		resource_id integer NOT NULL,
		user_id integer NOT NULL
	);
	
	
	ALTER TABLE public.handmade_libraryresourcestar OWNER TO hmn;
	
	--
	-- Name: handmade_libraryresourcestar_id_seq; Type: SEQUENCE; Schema: public; Owner: hmn
	--
	
	CREATE SEQUENCE public.handmade_libraryresourcestar_id_seq
		START WITH 1
		INCREMENT BY 1
		NO MINVALUE
		NO MAXVALUE
		CACHE 1;
	
	
	ALTER TABLE public.handmade_libraryresourcestar_id_seq OWNER TO hmn;
	
	--
	-- Name: handmade_libraryresourcestar_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: hmn
	--
	
	ALTER SEQUENCE public.handmade_libraryresourcestar_id_seq OWNED BY public.handmade_libraryresourcestar.id;
	
	
	--
	-- Name: handmade_librarytopic; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.handmade_librarytopic (
		id integer NOT NULL,
		name character varying(255) NOT NULL,
		description character varying(2000) NOT NULL,
		parent_id integer,
		project_id integer NOT NULL,
		is_root boolean NOT NULL
	);
	
	
	ALTER TABLE public.handmade_librarytopic OWNER TO hmn;
	
	--
	-- Name: handmade_librarytopic_id_seq; Type: SEQUENCE; Schema: public; Owner: hmn
	--
	
	CREATE SEQUENCE public.handmade_librarytopic_id_seq
		START WITH 1
		INCREMENT BY 1
		NO MINVALUE
		NO MAXVALUE
		CACHE 1;
	
	
	ALTER TABLE public.handmade_librarytopic_id_seq OWNER TO hmn;
	
	--
	-- Name: handmade_librarytopic_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: hmn
	--
	
	ALTER SEQUENCE public.handmade_librarytopic_id_seq OWNED BY public.handmade_librarytopic.id;
	
	
	--
	-- Name: handmade_license; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.handmade_license (
		slug character varying(255) NOT NULL,
		title character varying(255) NOT NULL
	);
	
	
	ALTER TABLE public.handmade_license OWNER TO hmn;
	
	--
	-- Name: handmade_license_texts; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.handmade_license_texts (
		id integer NOT NULL,
		license_id character varying(255) NOT NULL,
		post_id integer NOT NULL
	);
	
	
	ALTER TABLE public.handmade_license_texts OWNER TO hmn;
	
	--
	-- Name: handmade_license_texts_id_seq; Type: SEQUENCE; Schema: public; Owner: hmn
	--
	
	CREATE SEQUENCE public.handmade_license_texts_id_seq
		START WITH 1
		INCREMENT BY 1
		NO MINVALUE
		NO MAXVALUE
		CACHE 1;
	
	
	ALTER TABLE public.handmade_license_texts_id_seq OWNER TO hmn;
	
	--
	-- Name: handmade_license_texts_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: hmn
	--
	
	ALTER SEQUENCE public.handmade_license_texts_id_seq OWNED BY public.handmade_license_texts.id;
	
	
	--
	-- Name: handmade_links; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.handmade_links (
		id integer NOT NULL,
		key character varying(255) NOT NULL,
		name character varying(255),
		value character varying(255) NOT NULL,
		ordering integer NOT NULL
	);
	
	
	ALTER TABLE public.handmade_links OWNER TO hmn;
	
	--
	-- Name: handmade_links_id_seq; Type: SEQUENCE; Schema: public; Owner: hmn
	--
	
	CREATE SEQUENCE public.handmade_links_id_seq
		START WITH 1
		INCREMENT BY 1
		NO MINVALUE
		NO MAXVALUE
		CACHE 1;
	
	
	ALTER TABLE public.handmade_links_id_seq OWNER TO hmn;
	
	--
	-- Name: handmade_links_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: hmn
	--
	
	ALTER SEQUENCE public.handmade_links_id_seq OWNED BY public.handmade_links.id;
	
	
	--
	-- Name: handmade_member; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.handmade_member (
		user_id integer NOT NULL,
		blurb character varying(140),
		name character varying(255),
		signature text,
		avatar character varying(100),
		location character varying(100),
		ordering integer NOT NULL,
		posts integer NOT NULL,
		profileviews integer NOT NULL,
		thanked integer NOT NULL,
		timezone character varying(255) NOT NULL,
		color_1 character varying(6) NOT NULL,
		color_2 character varying(6) NOT NULL,
		darktheme boolean NOT NULL,
		extended_id integer NOT NULL,
		project_count_all integer NOT NULL,
		project_count_public integer NOT NULL,
		matrix_username character varying(255),
		set_matrix_display_name integer NOT NULL,
		edit_library boolean NOT NULL,
		discord_delete_snippet_on_message_delete boolean NOT NULL,
		discord_save_showcase boolean NOT NULL
	);
	
	
	ALTER TABLE public.handmade_member OWNER TO hmn;
	
	--
	-- Name: handmade_member_projects; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.handmade_member_projects (
		id integer NOT NULL,
		member_id integer NOT NULL,
		project_id integer NOT NULL
	);
	
	
	ALTER TABLE public.handmade_member_projects OWNER TO hmn;
	
	--
	-- Name: handmade_member_projects_id_seq; Type: SEQUENCE; Schema: public; Owner: hmn
	--
	
	CREATE SEQUENCE public.handmade_member_projects_id_seq
		START WITH 1
		INCREMENT BY 1
		NO MINVALUE
		NO MAXVALUE
		CACHE 1;
	
	
	ALTER TABLE public.handmade_member_projects_id_seq OWNER TO hmn;
	
	--
	-- Name: handmade_member_projects_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: hmn
	--
	
	ALTER SEQUENCE public.handmade_member_projects_id_seq OWNED BY public.handmade_member_projects.id;
	
	
	--
	-- Name: handmade_memberextended; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.handmade_memberextended (
		id integer NOT NULL,
		bio text,
		showemail boolean NOT NULL,
		sendemail integer NOT NULL,
		joomlaid integer,
		"lastResetTime" timestamp with time zone,
		"resetCount" integer NOT NULL,
		"requireReset" boolean NOT NULL,
		birthdate date
	);
	
	
	ALTER TABLE public.handmade_memberextended OWNER TO hmn;
	
	--
	-- Name: handmade_memberextended_id_seq; Type: SEQUENCE; Schema: public; Owner: hmn
	--
	
	CREATE SEQUENCE public.handmade_memberextended_id_seq
		START WITH 1
		INCREMENT BY 1
		NO MINVALUE
		NO MAXVALUE
		CACHE 1;
	
	
	ALTER TABLE public.handmade_memberextended_id_seq OWNER TO hmn;
	
	--
	-- Name: handmade_memberextended_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: hmn
	--
	
	ALTER SEQUENCE public.handmade_memberextended_id_seq OWNED BY public.handmade_memberextended.id;
	
	
	--
	-- Name: handmade_memberextended_links; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.handmade_memberextended_links (
		id integer NOT NULL,
		memberextended_id integer NOT NULL,
		links_id integer NOT NULL
	);
	
	
	ALTER TABLE public.handmade_memberextended_links OWNER TO hmn;
	
	--
	-- Name: handmade_memberextended_links_id_seq; Type: SEQUENCE; Schema: public; Owner: hmn
	--
	
	CREATE SEQUENCE public.handmade_memberextended_links_id_seq
		START WITH 1
		INCREMENT BY 1
		NO MINVALUE
		NO MAXVALUE
		CACHE 1;
	
	
	ALTER TABLE public.handmade_memberextended_links_id_seq OWNER TO hmn;
	
	--
	-- Name: handmade_memberextended_links_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: hmn
	--
	
	ALTER SEQUENCE public.handmade_memberextended_links_id_seq OWNED BY public.handmade_memberextended_links.id;
	
	
	--
	-- Name: handmade_onetimetoken; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.handmade_onetimetoken (
		id integer NOT NULL,
		token_type integer NOT NULL,
		created timestamp with time zone NOT NULL,
		used timestamp with time zone,
		expires timestamp with time zone,
		token_content character varying(100)
	);
	
	
	ALTER TABLE public.handmade_onetimetoken OWNER TO hmn;
	
	--
	-- Name: handmade_onetimetoken_id_seq; Type: SEQUENCE; Schema: public; Owner: hmn
	--
	
	CREATE SEQUENCE public.handmade_onetimetoken_id_seq
		START WITH 1
		INCREMENT BY 1
		NO MINVALUE
		NO MAXVALUE
		CACHE 1;
	
	
	ALTER TABLE public.handmade_onetimetoken_id_seq OWNER TO hmn;
	
	--
	-- Name: handmade_onetimetoken_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: hmn
	--
	
	ALTER SEQUENCE public.handmade_onetimetoken_id_seq OWNED BY public.handmade_onetimetoken.id;
	
	
	--
	-- Name: handmade_otherfile; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.handmade_otherfile (
		id integer NOT NULL,
		file character varying(255),
		size integer NOT NULL,
		sha1sum character varying(40) NOT NULL,
		protected boolean NOT NULL
	);
	
	
	ALTER TABLE public.handmade_otherfile OWNER TO hmn;
	
	--
	-- Name: handmade_otherfile_id_seq; Type: SEQUENCE; Schema: public; Owner: hmn
	--
	
	CREATE SEQUENCE public.handmade_otherfile_id_seq
		START WITH 1
		INCREMENT BY 1
		NO MINVALUE
		NO MAXVALUE
		CACHE 1;
	
	
	ALTER TABLE public.handmade_otherfile_id_seq OWNER TO hmn;
	
	--
	-- Name: handmade_otherfile_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: hmn
	--
	
	ALTER SEQUENCE public.handmade_otherfile_id_seq OWNED BY public.handmade_otherfile.id;
	
	
	--
	-- Name: handmade_passwordresetrequest; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.handmade_passwordresetrequest (
		id integer NOT NULL,
		confirmation_token_id integer NOT NULL,
		user_id integer NOT NULL
	);
	
	
	ALTER TABLE public.handmade_passwordresetrequest OWNER TO hmn;
	
	--
	-- Name: handmade_passwordresetrequest_id_seq; Type: SEQUENCE; Schema: public; Owner: hmn
	--
	
	CREATE SEQUENCE public.handmade_passwordresetrequest_id_seq
		START WITH 1
		INCREMENT BY 1
		NO MINVALUE
		NO MAXVALUE
		CACHE 1;
	
	
	ALTER TABLE public.handmade_passwordresetrequest_id_seq OWNER TO hmn;
	
	--
	-- Name: handmade_passwordresetrequest_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: hmn
	--
	
	ALTER SEQUENCE public.handmade_passwordresetrequest_id_seq OWNED BY public.handmade_passwordresetrequest.id;
	
	
	--
	-- Name: handmade_podcast; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.handmade_podcast (
		id integer NOT NULL,
		title character varying(255) NOT NULL,
		description character varying(4000) NOT NULL,
		language character varying(10) NOT NULL,
		image_id integer NOT NULL,
		project_id integer NOT NULL
	);
	
	
	ALTER TABLE public.handmade_podcast OWNER TO hmn;
	
	--
	-- Name: handmade_podcast_id_seq; Type: SEQUENCE; Schema: public; Owner: hmn
	--
	
	CREATE SEQUENCE public.handmade_podcast_id_seq
		START WITH 1
		INCREMENT BY 1
		NO MINVALUE
		NO MAXVALUE
		CACHE 1;
	
	
	ALTER TABLE public.handmade_podcast_id_seq OWNER TO hmn;
	
	--
	-- Name: handmade_podcast_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: hmn
	--
	
	ALTER SEQUENCE public.handmade_podcast_id_seq OWNED BY public.handmade_podcast.id;
	
	
	--
	-- Name: handmade_podcastepisode; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.handmade_podcastepisode (
		guid uuid NOT NULL,
		title character varying(255) NOT NULL,
		description character varying(4000) NOT NULL,
		"enclosureFile" character varying(255) NOT NULL,
		"pubDate" timestamp with time zone NOT NULL,
		duration integer NOT NULL,
		"episodeNumber" integer NOT NULL,
		"seasonNumber" integer,
		podcast_id integer NOT NULL,
		description_rendered character varying(4000) NOT NULL
	);
	
	
	ALTER TABLE public.handmade_podcastepisode OWNER TO hmn;
	
	--
	-- Name: handmade_post; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.handmade_post (
		id integer NOT NULL,
		depth integer NOT NULL,
		slug character varying(100) NOT NULL,
		author_name character varying(255) NOT NULL,
		postdate timestamp with time zone NOT NULL,
		ip inet NOT NULL,
		sticky boolean NOT NULL,
		moderated boolean NOT NULL,
		hits integer NOT NULL,
		featured boolean NOT NULL,
		featurevotes integer NOT NULL,
		author_id integer,
		category_id integer NOT NULL,
		parent_id integer,
		thread_id integer,
		preview character varying(100) NOT NULL,
		current_id integer NOT NULL,
		readonly boolean NOT NULL
	);
	
	
	ALTER TABLE public.handmade_post OWNER TO hmn;
	
	--
	-- Name: handmade_post_id_seq; Type: SEQUENCE; Schema: public; Owner: hmn
	--
	
	CREATE SEQUENCE public.handmade_post_id_seq
		START WITH 1
		INCREMENT BY 1
		NO MINVALUE
		NO MAXVALUE
		CACHE 1;
	
	
	ALTER TABLE public.handmade_post_id_seq OWNER TO hmn;
	
	--
	-- Name: handmade_post_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: hmn
	--
	
	ALTER SEQUENCE public.handmade_post_id_seq OWNED BY public.handmade_post.id;
	
	
	--
	-- Name: handmade_posttext; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.handmade_posttext (
		id integer NOT NULL,
		text text,
		textparsed text,
		parser integer NOT NULL
	);
	
	
	ALTER TABLE public.handmade_posttext OWNER TO hmn;
	
	--
	-- Name: handmade_posttext_id_seq; Type: SEQUENCE; Schema: public; Owner: hmn
	--
	
	CREATE SEQUENCE public.handmade_posttext_id_seq
		START WITH 1
		INCREMENT BY 1
		NO MINVALUE
		NO MAXVALUE
		CACHE 1;
	
	
	ALTER TABLE public.handmade_posttext_id_seq OWNER TO hmn;
	
	--
	-- Name: handmade_posttext_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: hmn
	--
	
	ALTER SEQUENCE public.handmade_posttext_id_seq OWNED BY public.handmade_posttext.id;
	
	
	--
	-- Name: handmade_posttextversion; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.handmade_posttextversion (
		id integer NOT NULL,
		title character varying(255) NOT NULL,
		editip inet,
		editdate timestamp with time zone,
		editreason character varying(255),
		editor_id integer,
		post_id integer,
		text_id integer NOT NULL
	);
	
	
	ALTER TABLE public.handmade_posttextversion OWNER TO hmn;
	
	--
	-- Name: handmade_posttextversion_id_seq; Type: SEQUENCE; Schema: public; Owner: hmn
	--
	
	CREATE SEQUENCE public.handmade_posttextversion_id_seq
		START WITH 1
		INCREMENT BY 1
		NO MINVALUE
		NO MAXVALUE
		CACHE 1;
	
	
	ALTER TABLE public.handmade_posttextversion_id_seq OWNER TO hmn;
	
	--
	-- Name: handmade_posttextversion_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: hmn
	--
	
	ALTER SEQUENCE public.handmade_posttextversion_id_seq OWNED BY public.handmade_posttextversion.id;
	
	
	--
	-- Name: handmade_project; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.handmade_project (
		id integer NOT NULL,
		slug character varying(30),
		name character varying(255),
		blurb character varying(140),
		description text,
		logodark character varying(100),
		background character varying(100),
		color_1 character varying(6) NOT NULL,
		color_2 character varying(6) NOT NULL,
		featured boolean NOT NULL,
		featurevotes integer NOT NULL,
		blog_id integer,
		forum_id integer,
		parent_id integer,
		standalone boolean NOT NULL,
		flags integer NOT NULL,
		annotation_id integer,
		static_id integer,
		logolight character varying(100),
		quota integer NOT NULL,
		quota_used integer NOT NULL,
		descparsed text,
		annotation_flags integer NOT NULL,
		blog_flags integer NOT NULL,
		forum_flags integer NOT NULL,
		static_flags integer NOT NULL,
		all_last_updated timestamp with time zone NOT NULL,
		annotation_last_updated timestamp with time zone NOT NULL,
		blog_last_updated timestamp with time zone NOT NULL,
		forum_last_updated timestamp with time zone NOT NULL,
		profile_last_updated timestamp with time zone NOT NULL,
		static_last_updated timestamp with time zone NOT NULL,
		lifecycle integer NOT NULL,
		date_approved timestamp with time zone NOT NULL,
		date_created timestamp with time zone NOT NULL,
		wiki_id integer,
		wiki_flags integer NOT NULL,
		wiki_last_updated timestamp with time zone NOT NULL,
		bg_flags integer NOT NULL,
		library_flags integer NOT NULL
	);
	
	
	ALTER TABLE public.handmade_project OWNER TO hmn;
	
	--
	-- Name: handmade_project_downloads; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.handmade_project_downloads (
		id integer NOT NULL,
		project_id integer NOT NULL,
		otherfile_id integer NOT NULL
	);
	
	
	ALTER TABLE public.handmade_project_downloads OWNER TO hmn;
	
	--
	-- Name: handmade_project_downloads_id_seq; Type: SEQUENCE; Schema: public; Owner: hmn
	--
	
	CREATE SEQUENCE public.handmade_project_downloads_id_seq
		START WITH 1
		INCREMENT BY 1
		NO MINVALUE
		NO MAXVALUE
		CACHE 1;
	
	
	ALTER TABLE public.handmade_project_downloads_id_seq OWNER TO hmn;
	
	--
	-- Name: handmade_project_downloads_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: hmn
	--
	
	ALTER SEQUENCE public.handmade_project_downloads_id_seq OWNED BY public.handmade_project_downloads.id;
	
	
	--
	-- Name: handmade_project_groups; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.handmade_project_groups (
		id integer NOT NULL,
		project_id integer NOT NULL,
		group_id integer NOT NULL
	);
	
	
	ALTER TABLE public.handmade_project_groups OWNER TO hmn;
	
	--
	-- Name: handmade_project_groups_id_seq; Type: SEQUENCE; Schema: public; Owner: hmn
	--
	
	CREATE SEQUENCE public.handmade_project_groups_id_seq
		START WITH 1
		INCREMENT BY 1
		NO MINVALUE
		NO MAXVALUE
		CACHE 1;
	
	
	ALTER TABLE public.handmade_project_groups_id_seq OWNER TO hmn;
	
	--
	-- Name: handmade_project_groups_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: hmn
	--
	
	ALTER SEQUENCE public.handmade_project_groups_id_seq OWNED BY public.handmade_project_groups.id;
	
	
	--
	-- Name: handmade_project_id_seq; Type: SEQUENCE; Schema: public; Owner: hmn
	--
	
	CREATE SEQUENCE public.handmade_project_id_seq
		START WITH 1
		INCREMENT BY 1
		NO MINVALUE
		NO MAXVALUE
		CACHE 1;
	
	
	ALTER TABLE public.handmade_project_id_seq OWNER TO hmn;
	
	--
	-- Name: handmade_project_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: hmn
	--
	
	ALTER SEQUENCE public.handmade_project_id_seq OWNED BY public.handmade_project.id;
	
	
	--
	-- Name: handmade_project_languages; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.handmade_project_languages (
		id integer NOT NULL,
		project_id integer NOT NULL,
		codelanguage_id integer NOT NULL
	);
	
	
	ALTER TABLE public.handmade_project_languages OWNER TO hmn;
	
	--
	-- Name: handmade_project_languages_id_seq; Type: SEQUENCE; Schema: public; Owner: hmn
	--
	
	CREATE SEQUENCE public.handmade_project_languages_id_seq
		START WITH 1
		INCREMENT BY 1
		NO MINVALUE
		NO MAXVALUE
		CACHE 1;
	
	
	ALTER TABLE public.handmade_project_languages_id_seq OWNER TO hmn;
	
	--
	-- Name: handmade_project_languages_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: hmn
	--
	
	ALTER SEQUENCE public.handmade_project_languages_id_seq OWNED BY public.handmade_project_languages.id;
	
	
	--
	-- Name: handmade_project_licenses; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.handmade_project_licenses (
		id integer NOT NULL,
		project_id integer NOT NULL,
		license_id character varying(255) NOT NULL
	);
	
	
	ALTER TABLE public.handmade_project_licenses OWNER TO hmn;
	
	--
	-- Name: handmade_project_licenses_id_seq; Type: SEQUENCE; Schema: public; Owner: hmn
	--
	
	CREATE SEQUENCE public.handmade_project_licenses_id_seq
		START WITH 1
		INCREMENT BY 1
		NO MINVALUE
		NO MAXVALUE
		CACHE 1;
	
	
	ALTER TABLE public.handmade_project_licenses_id_seq OWNER TO hmn;
	
	--
	-- Name: handmade_project_licenses_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: hmn
	--
	
	ALTER SEQUENCE public.handmade_project_licenses_id_seq OWNED BY public.handmade_project_licenses.id;
	
	
	--
	-- Name: handmade_project_links; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.handmade_project_links (
		id integer NOT NULL,
		project_id integer NOT NULL,
		links_id integer NOT NULL
	);
	
	
	ALTER TABLE public.handmade_project_links OWNER TO hmn;
	
	--
	-- Name: handmade_project_links_id_seq; Type: SEQUENCE; Schema: public; Owner: hmn
	--
	
	CREATE SEQUENCE public.handmade_project_links_id_seq
		START WITH 1
		INCREMENT BY 1
		NO MINVALUE
		NO MAXVALUE
		CACHE 1;
	
	
	ALTER TABLE public.handmade_project_links_id_seq OWNER TO hmn;
	
	--
	-- Name: handmade_project_links_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: hmn
	--
	
	ALTER SEQUENCE public.handmade_project_links_id_seq OWNED BY public.handmade_project_links.id;
	
	
	--
	-- Name: handmade_project_screenshots; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.handmade_project_screenshots (
		id integer NOT NULL,
		project_id integer NOT NULL,
		imagefile_id integer NOT NULL
	);
	
	
	ALTER TABLE public.handmade_project_screenshots OWNER TO hmn;
	
	--
	-- Name: handmade_project_screenshots_id_seq; Type: SEQUENCE; Schema: public; Owner: hmn
	--
	
	CREATE SEQUENCE public.handmade_project_screenshots_id_seq
		START WITH 1
		INCREMENT BY 1
		NO MINVALUE
		NO MAXVALUE
		CACHE 1;
	
	
	ALTER TABLE public.handmade_project_screenshots_id_seq OWNER TO hmn;
	
	--
	-- Name: handmade_project_screenshots_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: hmn
	--
	
	ALTER SEQUENCE public.handmade_project_screenshots_id_seq OWNED BY public.handmade_project_screenshots.id;
	
	
	--
	-- Name: handmade_snippet; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.handmade_snippet (
		id integer NOT NULL,
		url character varying(1000),
		description character varying(5000) NOT NULL,
		"when" timestamp with time zone NOT NULL,
		edited_on_website boolean NOT NULL,
		_description_html character varying(10000) NOT NULL,
		asset_id uuid,
		discord_message_id character varying(255),
		owner_id integer NOT NULL
	);
	
	
	ALTER TABLE public.handmade_snippet OWNER TO hmn;
	
	--
	-- Name: handmade_snippet_id_seq; Type: SEQUENCE; Schema: public; Owner: hmn
	--
	
	CREATE SEQUENCE public.handmade_snippet_id_seq
		START WITH 1
		INCREMENT BY 1
		NO MINVALUE
		NO MAXVALUE
		CACHE 1;
	
	
	ALTER TABLE public.handmade_snippet_id_seq OWNER TO hmn;
	
	--
	-- Name: handmade_snippet_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: hmn
	--
	
	ALTER SEQUENCE public.handmade_snippet_id_seq OWNED BY public.handmade_snippet.id;
	
	
	--
	-- Name: handmade_thread; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.handmade_thread (
		id integer NOT NULL,
		title character varying(255) NOT NULL,
		hits integer NOT NULL,
		reply_count integer NOT NULL,
		sticky boolean NOT NULL,
		locked boolean NOT NULL,
		moderated integer NOT NULL,
		category_id integer NOT NULL,
		first_id integer,
		last_id integer
	);
	
	
	ALTER TABLE public.handmade_thread OWNER TO hmn;
	
	--
	-- Name: handmade_thread_id_seq; Type: SEQUENCE; Schema: public; Owner: hmn
	--
	
	CREATE SEQUENCE public.handmade_thread_id_seq
		START WITH 1
		INCREMENT BY 1
		NO MINVALUE
		NO MAXVALUE
		CACHE 1;
	
	
	ALTER TABLE public.handmade_thread_id_seq OWNER TO hmn;
	
	--
	-- Name: handmade_thread_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: hmn
	--
	
	ALTER SEQUENCE public.handmade_thread_id_seq OWNED BY public.handmade_thread.id;
	
	
	--
	-- Name: handmade_threadlastreadinfo; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.handmade_threadlastreadinfo (
		id integer NOT NULL,
		lastread timestamp with time zone,
		member_id integer NOT NULL,
		category_id integer,
		thread_id integer
	);
	
	
	ALTER TABLE public.handmade_threadlastreadinfo OWNER TO hmn;
	
	--
	-- Name: handmade_threadlastreadinfo_id_seq; Type: SEQUENCE; Schema: public; Owner: hmn
	--
	
	CREATE SEQUENCE public.handmade_threadlastreadinfo_id_seq
		START WITH 1
		INCREMENT BY 1
		NO MINVALUE
		NO MAXVALUE
		CACHE 1;
	
	
	ALTER TABLE public.handmade_threadlastreadinfo_id_seq OWNER TO hmn;
	
	--
	-- Name: handmade_threadlastreadinfo_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: hmn
	--
	
	ALTER SEQUENCE public.handmade_threadlastreadinfo_id_seq OWNED BY public.handmade_threadlastreadinfo.id;
	
	
	--
	-- Name: handmade_userpending; Type: TABLE; Schema: public; Owner: hmn
	--
	
	CREATE TABLE public.handmade_userpending (
		id integer NOT NULL,
		status integer NOT NULL,
		username character varying(150) NOT NULL,
		first_name character varying(30),
		last_name character varying(30),
		email character varying(255) NOT NULL,
		password character varying(300) NOT NULL,
		date_joined timestamp with time zone NOT NULL,
		blurb character varying(140),
		name character varying(255),
		signature text,
		bio text,
		admin_note text,
		email_normalized character varying(255) NOT NULL,
		ip inet NOT NULL,
		useragent character varying(300) NOT NULL,
		referer character varying(300),
		comment text,
		website character varying(300),
		activation_token_id integer NOT NULL
	);
	
	
	ALTER TABLE public.handmade_userpending OWNER TO hmn;
	
	--
	-- Name: handmade_userpending_id_seq; Type: SEQUENCE; Schema: public; Owner: hmn
	--
	
	CREATE SEQUENCE public.handmade_userpending_id_seq
		START WITH 1
		INCREMENT BY 1
		NO MINVALUE
		NO MAXVALUE
		CACHE 1;
	
	
	ALTER TABLE public.handmade_userpending_id_seq OWNER TO hmn;
	
	--
	-- Name: handmade_userpending_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: hmn
	--
	
	ALTER SEQUENCE public.handmade_userpending_id_seq OWNED BY public.handmade_userpending.id;
	
	
	--
	-- Name: auth_group id; Type: DEFAULT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.auth_group ALTER COLUMN id SET DEFAULT nextval('public.auth_group_id_seq'::regclass);
	
	
	--
	-- Name: auth_group_permissions id; Type: DEFAULT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.auth_group_permissions ALTER COLUMN id SET DEFAULT nextval('public.auth_group_permissions_id_seq'::regclass);
	
	
	--
	-- Name: auth_permission id; Type: DEFAULT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.auth_permission ALTER COLUMN id SET DEFAULT nextval('public.auth_permission_id_seq'::regclass);
	
	
	--
	-- Name: auth_user id; Type: DEFAULT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.auth_user ALTER COLUMN id SET DEFAULT nextval('public.auth_user_id_seq'::regclass);
	
	
	--
	-- Name: auth_user_groups id; Type: DEFAULT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.auth_user_groups ALTER COLUMN id SET DEFAULT nextval('public.auth_user_groups_id_seq'::regclass);
	
	
	--
	-- Name: auth_user_user_permissions id; Type: DEFAULT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.auth_user_user_permissions ALTER COLUMN id SET DEFAULT nextval('public.auth_user_user_permissions_id_seq'::regclass);
	
	
	--
	-- Name: django_admin_log id; Type: DEFAULT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.django_admin_log ALTER COLUMN id SET DEFAULT nextval('public.django_admin_log_id_seq'::regclass);
	
	
	--
	-- Name: django_content_type id; Type: DEFAULT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.django_content_type ALTER COLUMN id SET DEFAULT nextval('public.django_content_type_id_seq'::regclass);
	
	
	--
	-- Name: django_migrations id; Type: DEFAULT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.django_migrations ALTER COLUMN id SET DEFAULT nextval('public.django_migrations_id_seq'::regclass);
	
	
	--
	-- Name: django_site id; Type: DEFAULT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.django_site ALTER COLUMN id SET DEFAULT nextval('public.django_site_id_seq'::regclass);
	
	
	--
	-- Name: handmade_category id; Type: DEFAULT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_category ALTER COLUMN id SET DEFAULT nextval('public.handmade_category_id_seq'::regclass);
	
	
	--
	-- Name: handmade_categorylastreadinfo id; Type: DEFAULT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_categorylastreadinfo ALTER COLUMN id SET DEFAULT nextval('public.handmade_categorylastreadinfo_id_seq'::regclass);
	
	
	--
	-- Name: handmade_codelanguage id; Type: DEFAULT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_codelanguage ALTER COLUMN id SET DEFAULT nextval('public.handmade_codelanguage_id_seq'::regclass);
	
	
	--
	-- Name: handmade_communicationchoice id; Type: DEFAULT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_communicationchoice ALTER COLUMN id SET DEFAULT nextval('public.handmade_communicationchoice_id_seq'::regclass);
	
	
	--
	-- Name: handmade_communicationchoicelist id; Type: DEFAULT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_communicationchoicelist ALTER COLUMN id SET DEFAULT nextval('public.handmade_communicationchoicelist_id_seq'::regclass);
	
	
	--
	-- Name: handmade_communicationsubcategory id; Type: DEFAULT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_communicationsubcategory ALTER COLUMN id SET DEFAULT nextval('public.handmade_communicationsubcategory_id_seq'::regclass);
	
	
	--
	-- Name: handmade_communicationsubthread id; Type: DEFAULT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_communicationsubthread ALTER COLUMN id SET DEFAULT nextval('public.handmade_communicationsubthread_id_seq'::regclass);
	
	
	--
	-- Name: handmade_discord id; Type: DEFAULT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_discord ALTER COLUMN id SET DEFAULT nextval('public.handmade_discord_id_seq'::regclass);
	
	
	--
	-- Name: handmade_discordmessageembed id; Type: DEFAULT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_discordmessageembed ALTER COLUMN id SET DEFAULT nextval('public.handmade_discordmessageembed_id_seq'::regclass);
	
	
	--
	-- Name: handmade_imagefile id; Type: DEFAULT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_imagefile ALTER COLUMN id SET DEFAULT nextval('public.handmade_imagefile_id_seq'::regclass);
	
	
	--
	-- Name: handmade_librarymediatype id; Type: DEFAULT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_librarymediatype ALTER COLUMN id SET DEFAULT nextval('public.handmade_librarymediatype_id_seq'::regclass);
	
	
	--
	-- Name: handmade_libraryresource id; Type: DEFAULT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_libraryresource ALTER COLUMN id SET DEFAULT nextval('public.handmade_libraryresource_id_seq'::regclass);
	
	
	--
	-- Name: handmade_libraryresource_media_types id; Type: DEFAULT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_libraryresource_media_types ALTER COLUMN id SET DEFAULT nextval('public.handmade_libraryresource_media_types_id_seq'::regclass);
	
	
	--
	-- Name: handmade_libraryresource_topics id; Type: DEFAULT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_libraryresource_topics ALTER COLUMN id SET DEFAULT nextval('public.handmade_libraryresource_topics_id_seq'::regclass);
	
	
	--
	-- Name: handmade_libraryresourcestar id; Type: DEFAULT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_libraryresourcestar ALTER COLUMN id SET DEFAULT nextval('public.handmade_libraryresourcestar_id_seq'::regclass);
	
	
	--
	-- Name: handmade_librarytopic id; Type: DEFAULT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_librarytopic ALTER COLUMN id SET DEFAULT nextval('public.handmade_librarytopic_id_seq'::regclass);
	
	
	--
	-- Name: handmade_license_texts id; Type: DEFAULT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_license_texts ALTER COLUMN id SET DEFAULT nextval('public.handmade_license_texts_id_seq'::regclass);
	
	
	--
	-- Name: handmade_links id; Type: DEFAULT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_links ALTER COLUMN id SET DEFAULT nextval('public.handmade_links_id_seq'::regclass);
	
	
	--
	-- Name: handmade_member_projects id; Type: DEFAULT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_member_projects ALTER COLUMN id SET DEFAULT nextval('public.handmade_member_projects_id_seq'::regclass);
	
	
	--
	-- Name: handmade_memberextended id; Type: DEFAULT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_memberextended ALTER COLUMN id SET DEFAULT nextval('public.handmade_memberextended_id_seq'::regclass);
	
	
	--
	-- Name: handmade_memberextended_links id; Type: DEFAULT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_memberextended_links ALTER COLUMN id SET DEFAULT nextval('public.handmade_memberextended_links_id_seq'::regclass);
	
	
	--
	-- Name: handmade_onetimetoken id; Type: DEFAULT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_onetimetoken ALTER COLUMN id SET DEFAULT nextval('public.handmade_onetimetoken_id_seq'::regclass);
	
	
	--
	-- Name: handmade_otherfile id; Type: DEFAULT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_otherfile ALTER COLUMN id SET DEFAULT nextval('public.handmade_otherfile_id_seq'::regclass);
	
	
	--
	-- Name: handmade_passwordresetrequest id; Type: DEFAULT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_passwordresetrequest ALTER COLUMN id SET DEFAULT nextval('public.handmade_passwordresetrequest_id_seq'::regclass);
	
	
	--
	-- Name: handmade_podcast id; Type: DEFAULT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_podcast ALTER COLUMN id SET DEFAULT nextval('public.handmade_podcast_id_seq'::regclass);
	
	
	--
	-- Name: handmade_post id; Type: DEFAULT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_post ALTER COLUMN id SET DEFAULT nextval('public.handmade_post_id_seq'::regclass);
	
	
	--
	-- Name: handmade_posttext id; Type: DEFAULT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_posttext ALTER COLUMN id SET DEFAULT nextval('public.handmade_posttext_id_seq'::regclass);
	
	
	--
	-- Name: handmade_posttextversion id; Type: DEFAULT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_posttextversion ALTER COLUMN id SET DEFAULT nextval('public.handmade_posttextversion_id_seq'::regclass);
	
	
	--
	-- Name: handmade_project id; Type: DEFAULT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_project ALTER COLUMN id SET DEFAULT nextval('public.handmade_project_id_seq'::regclass);
	
	
	--
	-- Name: handmade_project_downloads id; Type: DEFAULT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_project_downloads ALTER COLUMN id SET DEFAULT nextval('public.handmade_project_downloads_id_seq'::regclass);
	
	
	--
	-- Name: handmade_project_groups id; Type: DEFAULT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_project_groups ALTER COLUMN id SET DEFAULT nextval('public.handmade_project_groups_id_seq'::regclass);
	
	
	--
	-- Name: handmade_project_languages id; Type: DEFAULT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_project_languages ALTER COLUMN id SET DEFAULT nextval('public.handmade_project_languages_id_seq'::regclass);
	
	
	--
	-- Name: handmade_project_licenses id; Type: DEFAULT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_project_licenses ALTER COLUMN id SET DEFAULT nextval('public.handmade_project_licenses_id_seq'::regclass);
	
	
	--
	-- Name: handmade_project_links id; Type: DEFAULT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_project_links ALTER COLUMN id SET DEFAULT nextval('public.handmade_project_links_id_seq'::regclass);
	
	
	--
	-- Name: handmade_project_screenshots id; Type: DEFAULT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_project_screenshots ALTER COLUMN id SET DEFAULT nextval('public.handmade_project_screenshots_id_seq'::regclass);
	
	
	--
	-- Name: handmade_snippet id; Type: DEFAULT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_snippet ALTER COLUMN id SET DEFAULT nextval('public.handmade_snippet_id_seq'::regclass);
	
	
	--
	-- Name: handmade_thread id; Type: DEFAULT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_thread ALTER COLUMN id SET DEFAULT nextval('public.handmade_thread_id_seq'::regclass);
	
	
	--
	-- Name: handmade_threadlastreadinfo id; Type: DEFAULT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_threadlastreadinfo ALTER COLUMN id SET DEFAULT nextval('public.handmade_threadlastreadinfo_id_seq'::regclass);
	
	
	--
	-- Name: handmade_userpending id; Type: DEFAULT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_userpending ALTER COLUMN id SET DEFAULT nextval('public.handmade_userpending_id_seq'::regclass);
	
	
	--
	-- Name: auth_group auth_group_name_key; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.auth_group
		ADD CONSTRAINT auth_group_name_key UNIQUE (name);
	
	
	--
	-- Name: auth_group_permissions auth_group_permissions_group_id_0cd325b0_uniq; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.auth_group_permissions
		ADD CONSTRAINT auth_group_permissions_group_id_0cd325b0_uniq UNIQUE (group_id, permission_id);
	
	
	--
	-- Name: auth_group_permissions auth_group_permissions_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.auth_group_permissions
		ADD CONSTRAINT auth_group_permissions_pkey PRIMARY KEY (id);
	
	
	--
	-- Name: auth_group auth_group_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.auth_group
		ADD CONSTRAINT auth_group_pkey PRIMARY KEY (id);
	
	
	--
	-- Name: auth_permission auth_permission_content_type_id_01ab375a_uniq; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.auth_permission
		ADD CONSTRAINT auth_permission_content_type_id_01ab375a_uniq UNIQUE (content_type_id, codename);
	
	
	--
	-- Name: auth_permission auth_permission_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.auth_permission
		ADD CONSTRAINT auth_permission_pkey PRIMARY KEY (id);
	
	
	--
	-- Name: auth_user_groups auth_user_groups_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.auth_user_groups
		ADD CONSTRAINT auth_user_groups_pkey PRIMARY KEY (id);
	
	
	--
	-- Name: auth_user_groups auth_user_groups_user_id_94350c0c_uniq; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.auth_user_groups
		ADD CONSTRAINT auth_user_groups_user_id_94350c0c_uniq UNIQUE (user_id, group_id);
	
	
	--
	-- Name: auth_user auth_user_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.auth_user
		ADD CONSTRAINT auth_user_pkey PRIMARY KEY (id);
	
	
	--
	-- Name: auth_user_user_permissions auth_user_user_permissions_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.auth_user_user_permissions
		ADD CONSTRAINT auth_user_user_permissions_pkey PRIMARY KEY (id);
	
	
	--
	-- Name: auth_user_user_permissions auth_user_user_permissions_user_id_14a6b632_uniq; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.auth_user_user_permissions
		ADD CONSTRAINT auth_user_user_permissions_user_id_14a6b632_uniq UNIQUE (user_id, permission_id);
	
	
	--
	-- Name: auth_user auth_user_username_key; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.auth_user
		ADD CONSTRAINT auth_user_username_key UNIQUE (username);
	
	
	--
	-- Name: django_admin_log django_admin_log_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.django_admin_log
		ADD CONSTRAINT django_admin_log_pkey PRIMARY KEY (id);
	
	
	--
	-- Name: django_content_type django_content_type_app_label_76bd3d3b_uniq; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.django_content_type
		ADD CONSTRAINT django_content_type_app_label_76bd3d3b_uniq UNIQUE (app_label, model);
	
	
	--
	-- Name: django_content_type django_content_type_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.django_content_type
		ADD CONSTRAINT django_content_type_pkey PRIMARY KEY (id);
	
	
	--
	-- Name: django_migrations django_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.django_migrations
		ADD CONSTRAINT django_migrations_pkey PRIMARY KEY (id);
	
	
	--
	-- Name: django_session django_session_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.django_session
		ADD CONSTRAINT django_session_pkey PRIMARY KEY (session_key);
	
	
	--
	-- Name: django_site django_site_domain_a2e37b91_uniq; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.django_site
		ADD CONSTRAINT django_site_domain_a2e37b91_uniq UNIQUE (domain);
	
	
	--
	-- Name: django_site django_site_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.django_site
		ADD CONSTRAINT django_site_pkey PRIMARY KEY (id);
	
	
	--
	-- Name: handmade_asset handmade_asset_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_asset
		ADD CONSTRAINT handmade_asset_pkey PRIMARY KEY (id);
	
	
	--
	-- Name: handmade_blacklisthostname handmade_blacklistdomain_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_blacklisthostname
		ADD CONSTRAINT handmade_blacklistdomain_pkey PRIMARY KEY (hostname);
	
	
	--
	-- Name: handmade_blacklistemail handmade_blacklistemail_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_blacklistemail
		ADD CONSTRAINT handmade_blacklistemail_pkey PRIMARY KEY (email);
	
	
	--
	-- Name: handmade_category handmade_category_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_category
		ADD CONSTRAINT handmade_category_pkey PRIMARY KEY (id);
	
	
	--
	-- Name: handmade_categorylastreadinfo handmade_categorylastreadinfo_member_id_cce36783_uniq; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_categorylastreadinfo
		ADD CONSTRAINT handmade_categorylastreadinfo_member_id_cce36783_uniq UNIQUE (member_id, category_id);
	
	
	--
	-- Name: handmade_categorylastreadinfo handmade_categorylastreadinfo_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_categorylastreadinfo
		ADD CONSTRAINT handmade_categorylastreadinfo_pkey PRIMARY KEY (id);
	
	
	--
	-- Name: handmade_codelanguage handmade_codelanguage_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_codelanguage
		ADD CONSTRAINT handmade_codelanguage_pkey PRIMARY KEY (id);
	
	
	--
	-- Name: handmade_communicationchoice handmade_communicationchoice_member_id_6c1ba91a_uniq; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_communicationchoice
		ADD CONSTRAINT handmade_communicationchoice_member_id_6c1ba91a_uniq UNIQUE (member_id, option_id);
	
	
	--
	-- Name: handmade_communicationchoice handmade_communicationchoice_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_communicationchoice
		ADD CONSTRAINT handmade_communicationchoice_pkey PRIMARY KEY (id);
	
	
	--
	-- Name: handmade_communicationchoicelist handmade_communicationchoicelist_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_communicationchoicelist
		ADD CONSTRAINT handmade_communicationchoicelist_pkey PRIMARY KEY (id);
	
	
	--
	-- Name: handmade_communicationsubcategory handmade_communicationsubcategory_member_id_e3bb21d0_uniq; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_communicationsubcategory
		ADD CONSTRAINT handmade_communicationsubcategory_member_id_e3bb21d0_uniq UNIQUE (member_id, category_id);
	
	
	--
	-- Name: handmade_communicationsubcategory handmade_communicationsubcategory_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_communicationsubcategory
		ADD CONSTRAINT handmade_communicationsubcategory_pkey PRIMARY KEY (id);
	
	
	--
	-- Name: handmade_communicationsubthread handmade_communicationsubthread_member_id_41c80a7d_uniq; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_communicationsubthread
		ADD CONSTRAINT handmade_communicationsubthread_member_id_41c80a7d_uniq UNIQUE (member_id, thread_id);
	
	
	--
	-- Name: handmade_communicationsubthread handmade_communicationsubthread_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_communicationsubthread
		ADD CONSTRAINT handmade_communicationsubthread_pkey PRIMARY KEY (id);
	
	
	--
	-- Name: handmade_discord handmade_discord_member_id_key; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_discord
		ADD CONSTRAINT handmade_discord_member_id_key UNIQUE (member_id);
	
	
	--
	-- Name: handmade_discord handmade_discord_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_discord
		ADD CONSTRAINT handmade_discord_pkey PRIMARY KEY (id);
	
	
	--
	-- Name: handmade_discord handmade_discord_userid_96f6c49d_uniq; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_discord
		ADD CONSTRAINT handmade_discord_userid_96f6c49d_uniq UNIQUE (userid);
	
	
	--
	-- Name: handmade_discordmessage handmade_discordmessage_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_discordmessage
		ADD CONSTRAINT handmade_discordmessage_pkey PRIMARY KEY (id);
	
	
	--
	-- Name: handmade_discordmessageattachment handmade_discordmessageattachment_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_discordmessageattachment
		ADD CONSTRAINT handmade_discordmessageattachment_pkey PRIMARY KEY (id);
	
	
	--
	-- Name: handmade_discordmessagecontent handmade_discordmessagecontent_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_discordmessagecontent
		ADD CONSTRAINT handmade_discordmessagecontent_pkey PRIMARY KEY (message_id);
	
	
	--
	-- Name: handmade_discordmessageembed handmade_discordmessageembed_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_discordmessageembed
		ADD CONSTRAINT handmade_discordmessageembed_pkey PRIMARY KEY (id);
	
	
	--
	-- Name: handmade_imagefile handmade_imagefile_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_imagefile
		ADD CONSTRAINT handmade_imagefile_pkey PRIMARY KEY (id);
	
	
	--
	-- Name: handmade_kunenapost handmade_kunenapost_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_kunenapost
		ADD CONSTRAINT handmade_kunenapost_pkey PRIMARY KEY (kunenapost);
	
	
	--
	-- Name: handmade_kunenapost handmade_kunenapost_post_id_key; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_kunenapost
		ADD CONSTRAINT handmade_kunenapost_post_id_key UNIQUE (post_id);
	
	
	--
	-- Name: handmade_kunenathread handmade_kunenathread_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_kunenathread
		ADD CONSTRAINT handmade_kunenathread_pkey PRIMARY KEY (kunenathread);
	
	
	--
	-- Name: handmade_kunenathread handmade_kunenathread_thread_id_key; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_kunenathread
		ADD CONSTRAINT handmade_kunenathread_thread_id_key UNIQUE (thread_id);
	
	
	--
	-- Name: handmade_librarymediatype handmade_librarymediatype_name_f6466490_uniq; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_librarymediatype
		ADD CONSTRAINT handmade_librarymediatype_name_f6466490_uniq UNIQUE (name);
	
	
	--
	-- Name: handmade_librarymediatype handmade_librarymediatype_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_librarymediatype
		ADD CONSTRAINT handmade_librarymediatype_pkey PRIMARY KEY (id);
	
	
	--
	-- Name: handmade_libraryresource_media_types handmade_libraryresource_libraryresource_id_libra_661f15e0_uniq; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_libraryresource_media_types
		ADD CONSTRAINT handmade_libraryresource_libraryresource_id_libra_661f15e0_uniq UNIQUE (libraryresource_id, librarymediatype_id);
	
	
	--
	-- Name: handmade_libraryresource_topics handmade_libraryresource_libraryresource_id_libra_66d12f0c_uniq; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_libraryresource_topics
		ADD CONSTRAINT handmade_libraryresource_libraryresource_id_libra_66d12f0c_uniq UNIQUE (libraryresource_id, librarytopic_id);
	
	
	--
	-- Name: handmade_libraryresource_media_types handmade_libraryresource_media_types_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_libraryresource_media_types
		ADD CONSTRAINT handmade_libraryresource_media_types_pkey PRIMARY KEY (id);
	
	
	--
	-- Name: handmade_libraryresource handmade_libraryresource_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_libraryresource
		ADD CONSTRAINT handmade_libraryresource_pkey PRIMARY KEY (id);
	
	
	--
	-- Name: handmade_libraryresource_topics handmade_libraryresource_topics_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_libraryresource_topics
		ADD CONSTRAINT handmade_libraryresource_topics_pkey PRIMARY KEY (id);
	
	
	--
	-- Name: handmade_libraryresourcestar handmade_libraryresourcestar_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_libraryresourcestar
		ADD CONSTRAINT handmade_libraryresourcestar_pkey PRIMARY KEY (id);
	
	
	--
	-- Name: handmade_librarytopic handmade_librarytopic_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_librarytopic
		ADD CONSTRAINT handmade_librarytopic_pkey PRIMARY KEY (id);
	
	
	--
	-- Name: handmade_license handmade_license_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_license
		ADD CONSTRAINT handmade_license_pkey PRIMARY KEY (slug);
	
	
	--
	-- Name: handmade_license_texts handmade_license_texts_license_id_c7aa2e4a_uniq; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_license_texts
		ADD CONSTRAINT handmade_license_texts_license_id_c7aa2e4a_uniq UNIQUE (license_id, post_id);
	
	
	--
	-- Name: handmade_license_texts handmade_license_texts_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_license_texts
		ADD CONSTRAINT handmade_license_texts_pkey PRIMARY KEY (id);
	
	
	--
	-- Name: handmade_links handmade_links_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_links
		ADD CONSTRAINT handmade_links_pkey PRIMARY KEY (id);
	
	
	--
	-- Name: handmade_member handmade_member_extended_id_key; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_member
		ADD CONSTRAINT handmade_member_extended_id_key UNIQUE (extended_id);
	
	
	--
	-- Name: handmade_member handmade_member_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_member
		ADD CONSTRAINT handmade_member_pkey PRIMARY KEY (user_id);
	
	
	--
	-- Name: handmade_member_projects handmade_member_projects_member_id_d87f692e_uniq; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_member_projects
		ADD CONSTRAINT handmade_member_projects_member_id_d87f692e_uniq UNIQUE (member_id, project_id);
	
	
	--
	-- Name: handmade_member_projects handmade_member_projects_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_member_projects
		ADD CONSTRAINT handmade_member_projects_pkey PRIMARY KEY (id);
	
	
	--
	-- Name: handmade_memberextended handmade_memberextended_joomlaid_key; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_memberextended
		ADD CONSTRAINT handmade_memberextended_joomlaid_key UNIQUE (joomlaid);
	
	
	--
	-- Name: handmade_memberextended_links handmade_memberextended_links_memberextended_id_15e19b65_uniq; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_memberextended_links
		ADD CONSTRAINT handmade_memberextended_links_memberextended_id_15e19b65_uniq UNIQUE (memberextended_id, links_id);
	
	
	--
	-- Name: handmade_memberextended_links handmade_memberextended_links_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_memberextended_links
		ADD CONSTRAINT handmade_memberextended_links_pkey PRIMARY KEY (id);
	
	
	--
	-- Name: handmade_memberextended handmade_memberextended_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_memberextended
		ADD CONSTRAINT handmade_memberextended_pkey PRIMARY KEY (id);
	
	
	--
	-- Name: handmade_onetimetoken handmade_onetimetoken_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_onetimetoken
		ADD CONSTRAINT handmade_onetimetoken_pkey PRIMARY KEY (id);
	
	
	--
	-- Name: handmade_otherfile handmade_otherfile_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_otherfile
		ADD CONSTRAINT handmade_otherfile_pkey PRIMARY KEY (id);
	
	
	--
	-- Name: handmade_passwordresetrequest handmade_passwordresetrequest_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_passwordresetrequest
		ADD CONSTRAINT handmade_passwordresetrequest_pkey PRIMARY KEY (id);
	
	
	--
	-- Name: handmade_podcast handmade_podcast_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_podcast
		ADD CONSTRAINT handmade_podcast_pkey PRIMARY KEY (id);
	
	
	--
	-- Name: handmade_podcastepisode handmade_podcastepisode_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_podcastepisode
		ADD CONSTRAINT handmade_podcastepisode_pkey PRIMARY KEY (guid);
	
	
	--
	-- Name: handmade_post handmade_post_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_post
		ADD CONSTRAINT handmade_post_pkey PRIMARY KEY (id);
	
	
	--
	-- Name: handmade_posttext handmade_posttext_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_posttext
		ADD CONSTRAINT handmade_posttext_pkey PRIMARY KEY (id);
	
	
	--
	-- Name: handmade_posttextversion handmade_posttextversion_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_posttextversion
		ADD CONSTRAINT handmade_posttextversion_pkey PRIMARY KEY (id);
	
	
	--
	-- Name: handmade_project_downloads handmade_project_downloads_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_project_downloads
		ADD CONSTRAINT handmade_project_downloads_pkey PRIMARY KEY (id);
	
	
	--
	-- Name: handmade_project_downloads handmade_project_downloads_project_id_d84ef6f9_uniq; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_project_downloads
		ADD CONSTRAINT handmade_project_downloads_project_id_d84ef6f9_uniq UNIQUE (project_id, otherfile_id);
	
	
	--
	-- Name: handmade_project_groups handmade_project_groups_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_project_groups
		ADD CONSTRAINT handmade_project_groups_pkey PRIMARY KEY (id);
	
	
	--
	-- Name: handmade_project_groups handmade_project_groups_project_id_a5e3b4c5_uniq; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_project_groups
		ADD CONSTRAINT handmade_project_groups_project_id_a5e3b4c5_uniq UNIQUE (project_id, group_id);
	
	
	--
	-- Name: handmade_project_languages handmade_project_languages_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_project_languages
		ADD CONSTRAINT handmade_project_languages_pkey PRIMARY KEY (id);
	
	
	--
	-- Name: handmade_project_languages handmade_project_languages_project_id_74313c14_uniq; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_project_languages
		ADD CONSTRAINT handmade_project_languages_project_id_74313c14_uniq UNIQUE (project_id, codelanguage_id);
	
	
	--
	-- Name: handmade_project_licenses handmade_project_licenses_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_project_licenses
		ADD CONSTRAINT handmade_project_licenses_pkey PRIMARY KEY (id);
	
	
	--
	-- Name: handmade_project_licenses handmade_project_licenses_project_id_3924957a_uniq; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_project_licenses
		ADD CONSTRAINT handmade_project_licenses_project_id_3924957a_uniq UNIQUE (project_id, license_id);
	
	
	--
	-- Name: handmade_project_links handmade_project_links_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_project_links
		ADD CONSTRAINT handmade_project_links_pkey PRIMARY KEY (id);
	
	
	--
	-- Name: handmade_project_links handmade_project_links_project_id_790a99ea_uniq; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_project_links
		ADD CONSTRAINT handmade_project_links_project_id_790a99ea_uniq UNIQUE (project_id, links_id);
	
	
	--
	-- Name: handmade_project handmade_project_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_project
		ADD CONSTRAINT handmade_project_pkey PRIMARY KEY (id);
	
	
	--
	-- Name: handmade_project_screenshots handmade_project_screenshots_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_project_screenshots
		ADD CONSTRAINT handmade_project_screenshots_pkey PRIMARY KEY (id);
	
	
	--
	-- Name: handmade_project_screenshots handmade_project_screenshots_project_id_8b2eb536_uniq; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_project_screenshots
		ADD CONSTRAINT handmade_project_screenshots_project_id_8b2eb536_uniq UNIQUE (project_id, imagefile_id);
	
	
	--
	-- Name: handmade_snippet handmade_snippet_discord_message_id_key; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_snippet
		ADD CONSTRAINT handmade_snippet_discord_message_id_key UNIQUE (discord_message_id);
	
	
	--
	-- Name: handmade_snippet handmade_snippet_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_snippet
		ADD CONSTRAINT handmade_snippet_pkey PRIMARY KEY (id);
	
	
	--
	-- Name: handmade_thread handmade_thread_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_thread
		ADD CONSTRAINT handmade_thread_pkey PRIMARY KEY (id);
	
	
	--
	-- Name: handmade_threadlastreadinfo handmade_threadlastreadinfo_member_id_8c66fea8_uniq; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_threadlastreadinfo
		ADD CONSTRAINT handmade_threadlastreadinfo_member_id_8c66fea8_uniq UNIQUE (member_id, thread_id);
	
	
	--
	-- Name: handmade_threadlastreadinfo handmade_threadlastreadinfo_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_threadlastreadinfo
		ADD CONSTRAINT handmade_threadlastreadinfo_pkey PRIMARY KEY (id);
	
	
	--
	-- Name: handmade_userpending handmade_userpending_pkey; Type: CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_userpending
		ADD CONSTRAINT handmade_userpending_pkey PRIMARY KEY (id);
	
	
	--
	-- Name: auth_group_name_a6ea08ec_like; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX auth_group_name_a6ea08ec_like ON public.auth_group USING btree (name varchar_pattern_ops);
	
	
	--
	-- Name: auth_group_permissions_0e939a4f; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX auth_group_permissions_0e939a4f ON public.auth_group_permissions USING btree (group_id);
	
	
	--
	-- Name: auth_group_permissions_8373b171; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX auth_group_permissions_8373b171 ON public.auth_group_permissions USING btree (permission_id);
	
	
	--
	-- Name: auth_permission_417f1b1c; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX auth_permission_417f1b1c ON public.auth_permission USING btree (content_type_id);
	
	
	--
	-- Name: auth_user_groups_0e939a4f; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX auth_user_groups_0e939a4f ON public.auth_user_groups USING btree (group_id);
	
	
	--
	-- Name: auth_user_groups_e8701ad4; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX auth_user_groups_e8701ad4 ON public.auth_user_groups USING btree (user_id);
	
	
	--
	-- Name: auth_user_user_permissions_8373b171; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX auth_user_user_permissions_8373b171 ON public.auth_user_user_permissions USING btree (permission_id);
	
	
	--
	-- Name: auth_user_user_permissions_e8701ad4; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX auth_user_user_permissions_e8701ad4 ON public.auth_user_user_permissions USING btree (user_id);
	
	
	--
	-- Name: auth_user_username_6821ab7c_like; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX auth_user_username_6821ab7c_like ON public.auth_user USING btree (username varchar_pattern_ops);
	
	
	--
	-- Name: django_admin_log_417f1b1c; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX django_admin_log_417f1b1c ON public.django_admin_log USING btree (content_type_id);
	
	
	--
	-- Name: django_admin_log_e8701ad4; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX django_admin_log_e8701ad4 ON public.django_admin_log USING btree (user_id);
	
	
	--
	-- Name: django_session_de54fa62; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX django_session_de54fa62 ON public.django_session USING btree (expire_date);
	
	
	--
	-- Name: django_session_session_key_c0390e0f_like; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX django_session_session_key_c0390e0f_like ON public.django_session USING btree (session_key varchar_pattern_ops);
	
	
	--
	-- Name: django_site_domain_a2e37b91_like; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX django_site_domain_a2e37b91_like ON public.django_site USING btree (domain varchar_pattern_ops);
	
	
	--
	-- Name: handmade_asset_uploader_id_fc1cb702; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_asset_uploader_id_fc1cb702 ON public.handmade_asset USING btree (uploader_id);
	
	
	--
	-- Name: handmade_blacklistdomain_domain_e45fec72_like; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_blacklistdomain_domain_e45fec72_like ON public.handmade_blacklisthostname USING btree (hostname varchar_pattern_ops);
	
	
	--
	-- Name: handmade_blacklistemail_email_935c9fc5_like; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_blacklistemail_email_935c9fc5_like ON public.handmade_blacklistemail USING btree (email varchar_pattern_ops);
	
	
	--
	-- Name: handmade_category_6be37982; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_category_6be37982 ON public.handmade_category USING btree (parent_id);
	
	
	--
	-- Name: handmade_category_b098ad43; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_category_b098ad43 ON public.handmade_category USING btree (project_id);
	
	
	--
	-- Name: handmade_categorylastreadinfo_b583a629; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_categorylastreadinfo_b583a629 ON public.handmade_categorylastreadinfo USING btree (category_id);
	
	
	--
	-- Name: handmade_categorylastreadinfo_b5c3e75b; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_categorylastreadinfo_b5c3e75b ON public.handmade_categorylastreadinfo USING btree (member_id);
	
	
	--
	-- Name: handmade_categorylastreadinfo_member_id_cce36783_idx; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_categorylastreadinfo_member_id_cce36783_idx ON public.handmade_categorylastreadinfo USING btree (member_id, category_id);
	
	
	--
	-- Name: handmade_codelanguage_2dbcba41; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_codelanguage_2dbcba41 ON public.handmade_codelanguage USING btree (slug);
	
	
	--
	-- Name: handmade_codelanguage_slug_751774ab_like; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_codelanguage_slug_751774ab_like ON public.handmade_codelanguage USING btree (slug varchar_pattern_ops);
	
	
	--
	-- Name: handmade_communicationchoice_28df3725; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_communicationchoice_28df3725 ON public.handmade_communicationchoice USING btree (option_id);
	
	
	--
	-- Name: handmade_communicationchoice_b5c3e75b; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_communicationchoice_b5c3e75b ON public.handmade_communicationchoice USING btree (member_id);
	
	
	--
	-- Name: handmade_communicationchoice_member_id_6c1ba91a_idx; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_communicationchoice_member_id_6c1ba91a_idx ON public.handmade_communicationchoice USING btree (member_id, option_id);
	
	
	--
	-- Name: handmade_communicationchoicelist_3c6e0b8a; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_communicationchoicelist_3c6e0b8a ON public.handmade_communicationchoicelist USING btree (key);
	
	
	--
	-- Name: handmade_communicationchoicelist_b098ad43; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_communicationchoicelist_b098ad43 ON public.handmade_communicationchoicelist USING btree (project_id);
	
	
	--
	-- Name: handmade_communicationchoicelist_project_id_1343ebb4_idx; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_communicationchoicelist_project_id_1343ebb4_idx ON public.handmade_communicationchoicelist USING btree (project_id, key);
	
	
	--
	-- Name: handmade_communicationsubcategory_b583a629; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_communicationsubcategory_b583a629 ON public.handmade_communicationsubcategory USING btree (category_id);
	
	
	--
	-- Name: handmade_communicationsubcategory_b5c3e75b; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_communicationsubcategory_b5c3e75b ON public.handmade_communicationsubcategory USING btree (member_id);
	
	
	--
	-- Name: handmade_communicationsubcategory_member_id_e3bb21d0_idx; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_communicationsubcategory_member_id_e3bb21d0_idx ON public.handmade_communicationsubcategory USING btree (member_id, category_id);
	
	
	--
	-- Name: handmade_communicationsubthread_b5c3e75b; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_communicationsubthread_b5c3e75b ON public.handmade_communicationsubthread USING btree (member_id);
	
	
	--
	-- Name: handmade_communicationsubthread_e3464c97; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_communicationsubthread_e3464c97 ON public.handmade_communicationsubthread USING btree (thread_id);
	
	
	--
	-- Name: handmade_communicationsubthread_member_id_41c80a7d_idx; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_communicationsubthread_member_id_41c80a7d_idx ON public.handmade_communicationsubthread USING btree (member_id, thread_id);
	
	
	--
	-- Name: handmade_discord_userid_96f6c49d_like; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_discord_userid_96f6c49d_like ON public.handmade_discord USING btree (userid varchar_pattern_ops);
	
	
	--
	-- Name: handmade_discordmessage_id_8f029f93_like; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_discordmessage_id_8f029f93_like ON public.handmade_discordmessage USING btree (id varchar_pattern_ops);
	
	
	--
	-- Name: handmade_discordmessageattachment_asset_id_c64a3c31; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_discordmessageattachment_asset_id_c64a3c31 ON public.handmade_discordmessageattachment USING btree (asset_id);
	
	
	--
	-- Name: handmade_discordmessageattachment_id_2729be08_like; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_discordmessageattachment_id_2729be08_like ON public.handmade_discordmessageattachment USING btree (id varchar_pattern_ops);
	
	
	--
	-- Name: handmade_discordmessageattachment_message_id_d39da9b3; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_discordmessageattachment_message_id_d39da9b3 ON public.handmade_discordmessageattachment USING btree (message_id);
	
	
	--
	-- Name: handmade_discordmessageattachment_message_id_d39da9b3_like; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_discordmessageattachment_message_id_d39da9b3_like ON public.handmade_discordmessageattachment USING btree (message_id varchar_pattern_ops);
	
	
	--
	-- Name: handmade_discordmessagecontent_discord_id_1acc147f; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_discordmessagecontent_discord_id_1acc147f ON public.handmade_discordmessagecontent USING btree (discord_id);
	
	
	--
	-- Name: handmade_discordmessagecontent_message_id_4dfde67d_like; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_discordmessagecontent_message_id_4dfde67d_like ON public.handmade_discordmessagecontent USING btree (message_id varchar_pattern_ops);
	
	
	--
	-- Name: handmade_discordmessageembed_image_id_9b04bb5f; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_discordmessageembed_image_id_9b04bb5f ON public.handmade_discordmessageembed USING btree (image_id);
	
	
	--
	-- Name: handmade_discordmessageembed_message_id_04f15ce6; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_discordmessageembed_message_id_04f15ce6 ON public.handmade_discordmessageembed USING btree (message_id);
	
	
	--
	-- Name: handmade_discordmessageembed_message_id_04f15ce6_like; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_discordmessageembed_message_id_04f15ce6_like ON public.handmade_discordmessageembed USING btree (message_id varchar_pattern_ops);
	
	
	--
	-- Name: handmade_discordmessageembed_video_id_1c41289f; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_discordmessageembed_video_id_1c41289f ON public.handmade_discordmessageembed USING btree (video_id);
	
	
	--
	-- Name: handmade_librarymediatype_name_f6466490_like; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_librarymediatype_name_f6466490_like ON public.handmade_librarymediatype USING btree (name varchar_pattern_ops);
	
	
	--
	-- Name: handmade_libraryresource_category_id_06d45134; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_libraryresource_category_id_06d45134 ON public.handmade_libraryresource USING btree (category_id);
	
	
	--
	-- Name: handmade_libraryresource_m_librarymediatype_id_9589f3f4; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_libraryresource_m_librarymediatype_id_9589f3f4 ON public.handmade_libraryresource_media_types USING btree (librarymediatype_id);
	
	
	--
	-- Name: handmade_libraryresource_m_libraryresource_id_f643f15f; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_libraryresource_m_libraryresource_id_f643f15f ON public.handmade_libraryresource_media_types USING btree (libraryresource_id);
	
	
	--
	-- Name: handmade_libraryresource_project_id_a3d6e0fe; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_libraryresource_project_id_a3d6e0fe ON public.handmade_libraryresource USING btree (project_id);
	
	
	--
	-- Name: handmade_libraryresource_topics_libraryresource_id_74e35c80; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_libraryresource_topics_libraryresource_id_74e35c80 ON public.handmade_libraryresource_topics USING btree (libraryresource_id);
	
	
	--
	-- Name: handmade_libraryresource_topics_librarytopic_id_7aa70be0; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_libraryresource_topics_librarytopic_id_7aa70be0 ON public.handmade_libraryresource_topics USING btree (librarytopic_id);
	
	
	--
	-- Name: handmade_libraryresourcestar_resource_id_5db93928; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_libraryresourcestar_resource_id_5db93928 ON public.handmade_libraryresourcestar USING btree (resource_id);
	
	
	--
	-- Name: handmade_libraryresourcestar_user_id_b8483e28; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_libraryresourcestar_user_id_b8483e28 ON public.handmade_libraryresourcestar USING btree (user_id);
	
	
	--
	-- Name: handmade_librarytopic_parent_id_5dfddf8e; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_librarytopic_parent_id_5dfddf8e ON public.handmade_librarytopic USING btree (parent_id);
	
	
	--
	-- Name: handmade_librarytopic_project_id_3b1879da; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_librarytopic_project_id_3b1879da ON public.handmade_librarytopic USING btree (project_id);
	
	
	--
	-- Name: handmade_license_slug_db9c395a_like; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_license_slug_db9c395a_like ON public.handmade_license USING btree (slug varchar_pattern_ops);
	
	
	--
	-- Name: handmade_license_texts_366393cd; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_license_texts_366393cd ON public.handmade_license_texts USING btree (license_id);
	
	
	--
	-- Name: handmade_license_texts_f3aa1999; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_license_texts_f3aa1999 ON public.handmade_license_texts USING btree (post_id);
	
	
	--
	-- Name: handmade_license_texts_license_id_93d0ac5d_like; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_license_texts_license_id_93d0ac5d_like ON public.handmade_license_texts USING btree (license_id varchar_pattern_ops);
	
	
	--
	-- Name: handmade_member_projects_b098ad43; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_member_projects_b098ad43 ON public.handmade_member_projects USING btree (project_id);
	
	
	--
	-- Name: handmade_member_projects_b5c3e75b; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_member_projects_b5c3e75b ON public.handmade_member_projects USING btree (member_id);
	
	
	--
	-- Name: handmade_memberextended_links_0f48623a; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_memberextended_links_0f48623a ON public.handmade_memberextended_links USING btree (memberextended_id);
	
	
	--
	-- Name: handmade_memberextended_links_8de4cd5a; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_memberextended_links_8de4cd5a ON public.handmade_memberextended_links USING btree (links_id);
	
	
	--
	-- Name: handmade_passwordresetrequest_e88135c0; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_passwordresetrequest_e88135c0 ON public.handmade_passwordresetrequest USING btree (confirmation_token_id);
	
	
	--
	-- Name: handmade_podcast_image_id_cfbd1a68; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_podcast_image_id_cfbd1a68 ON public.handmade_podcast USING btree (image_id);
	
	
	--
	-- Name: handmade_podcast_project_id_bf27fb3a; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_podcast_project_id_bf27fb3a ON public.handmade_podcast USING btree (project_id);
	
	
	--
	-- Name: handmade_podcastepisode_podcast_id_b86d4941; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_podcastepisode_podcast_id_b86d4941 ON public.handmade_podcastepisode USING btree (podcast_id);
	
	
	--
	-- Name: handmade_post_2dbcba41; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_post_2dbcba41 ON public.handmade_post USING btree (slug);
	
	
	--
	-- Name: handmade_post_4f331e2f; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_post_4f331e2f ON public.handmade_post USING btree (author_id);
	
	
	--
	-- Name: handmade_post_6be37982; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_post_6be37982 ON public.handmade_post USING btree (parent_id);
	
	
	--
	-- Name: handmade_post_b583a629; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_post_b583a629 ON public.handmade_post USING btree (category_id);
	
	
	--
	-- Name: handmade_post_current_id_762211b7; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_post_current_id_762211b7 ON public.handmade_post USING btree (current_id);
	
	
	--
	-- Name: handmade_post_e3464c97; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_post_e3464c97 ON public.handmade_post USING btree (thread_id);
	
	
	--
	-- Name: handmade_post_slug_93476abd_like; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_post_slug_93476abd_like ON public.handmade_post USING btree (slug varchar_pattern_ops);
	
	
	--
	-- Name: handmade_posttextversion_editor_id_62fdd463; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_posttextversion_editor_id_62fdd463 ON public.handmade_posttextversion USING btree (editor_id);
	
	
	--
	-- Name: handmade_posttextversion_post_id_440a419c; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_posttextversion_post_id_440a419c ON public.handmade_posttextversion USING btree (post_id);
	
	
	--
	-- Name: handmade_posttextversion_text_id_4e0fde60; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_posttextversion_text_id_4e0fde60 ON public.handmade_posttextversion USING btree (text_id);
	
	
	--
	-- Name: handmade_project_19bc3ff1; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_project_19bc3ff1 ON public.handmade_project USING btree (forum_id);
	
	
	--
	-- Name: handmade_project_64458f32; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_project_64458f32 ON public.handmade_project USING btree (blog_id);
	
	
	--
	-- Name: handmade_project_6be37982; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_project_6be37982 ON public.handmade_project USING btree (parent_id);
	
	
	--
	-- Name: handmade_project_a33d2ace; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_project_a33d2ace ON public.handmade_project USING btree (static_id);
	
	
	--
	-- Name: handmade_project_downloads_474107cb; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_project_downloads_474107cb ON public.handmade_project_downloads USING btree (otherfile_id);
	
	
	--
	-- Name: handmade_project_downloads_b098ad43; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_project_downloads_b098ad43 ON public.handmade_project_downloads USING btree (project_id);
	
	
	--
	-- Name: handmade_project_e70a1874; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_project_e70a1874 ON public.handmade_project USING btree (annotation_id);
	
	
	--
	-- Name: handmade_project_groups_0e939a4f; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_project_groups_0e939a4f ON public.handmade_project_groups USING btree (group_id);
	
	
	--
	-- Name: handmade_project_groups_b098ad43; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_project_groups_b098ad43 ON public.handmade_project_groups USING btree (project_id);
	
	
	--
	-- Name: handmade_project_languages_a1b06203; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_project_languages_a1b06203 ON public.handmade_project_languages USING btree (codelanguage_id);
	
	
	--
	-- Name: handmade_project_languages_b098ad43; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_project_languages_b098ad43 ON public.handmade_project_languages USING btree (project_id);
	
	
	--
	-- Name: handmade_project_licenses_366393cd; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_project_licenses_366393cd ON public.handmade_project_licenses USING btree (license_id);
	
	
	--
	-- Name: handmade_project_licenses_b098ad43; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_project_licenses_b098ad43 ON public.handmade_project_licenses USING btree (project_id);
	
	
	--
	-- Name: handmade_project_licenses_license_id_618488c2_like; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_project_licenses_license_id_618488c2_like ON public.handmade_project_licenses USING btree (license_id varchar_pattern_ops);
	
	
	--
	-- Name: handmade_project_links_8de4cd5a; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_project_links_8de4cd5a ON public.handmade_project_links USING btree (links_id);
	
	
	--
	-- Name: handmade_project_links_b098ad43; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_project_links_b098ad43 ON public.handmade_project_links USING btree (project_id);
	
	
	--
	-- Name: handmade_project_screenshots_862beb90; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_project_screenshots_862beb90 ON public.handmade_project_screenshots USING btree (imagefile_id);
	
	
	--
	-- Name: handmade_project_screenshots_b098ad43; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_project_screenshots_b098ad43 ON public.handmade_project_screenshots USING btree (project_id);
	
	
	--
	-- Name: handmade_project_wiki_id_eba06ae0; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_project_wiki_id_eba06ae0 ON public.handmade_project USING btree (wiki_id);
	
	
	--
	-- Name: handmade_snippet_asset_id_c786de4f; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_snippet_asset_id_c786de4f ON public.handmade_snippet USING btree (asset_id);
	
	
	--
	-- Name: handmade_snippet_discord_message_id_d16f1f4e_like; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_snippet_discord_message_id_d16f1f4e_like ON public.handmade_snippet USING btree (discord_message_id varchar_pattern_ops);
	
	
	--
	-- Name: handmade_snippet_owner_id_fcca1783; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_snippet_owner_id_fcca1783 ON public.handmade_snippet USING btree (owner_id);
	
	
	--
	-- Name: handmade_thread_6dfe7580; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_thread_6dfe7580 ON public.handmade_thread USING btree (first_id);
	
	
	--
	-- Name: handmade_thread_70cbd484; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_thread_70cbd484 ON public.handmade_thread USING btree (last_id);
	
	
	--
	-- Name: handmade_thread_b583a629; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_thread_b583a629 ON public.handmade_thread USING btree (category_id);
	
	
	--
	-- Name: handmade_threadlastreadinfo_b583a629; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_threadlastreadinfo_b583a629 ON public.handmade_threadlastreadinfo USING btree (category_id);
	
	
	--
	-- Name: handmade_threadlastreadinfo_b5c3e75b; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_threadlastreadinfo_b5c3e75b ON public.handmade_threadlastreadinfo USING btree (member_id);
	
	
	--
	-- Name: handmade_threadlastreadinfo_e3464c97; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_threadlastreadinfo_e3464c97 ON public.handmade_threadlastreadinfo USING btree (thread_id);
	
	
	--
	-- Name: handmade_threadlastreadinfo_member_id_8c66fea8_idx; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_threadlastreadinfo_member_id_8c66fea8_idx ON public.handmade_threadlastreadinfo USING btree (member_id, thread_id);
	
	
	--
	-- Name: handmade_userpending_3acfbef1; Type: INDEX; Schema: public; Owner: hmn
	--
	
	CREATE INDEX handmade_userpending_3acfbef1 ON public.handmade_userpending USING btree (activation_token_id);
	
	
	--
	-- Name: auth_group_permissions auth_group_permiss_permission_id_84c5c92e_fk_auth_permission_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.auth_group_permissions
		ADD CONSTRAINT auth_group_permiss_permission_id_84c5c92e_fk_auth_permission_id FOREIGN KEY (permission_id) REFERENCES public.auth_permission(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: auth_group_permissions auth_group_permissions_group_id_b120cbf9_fk_auth_group_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.auth_group_permissions
		ADD CONSTRAINT auth_group_permissions_group_id_b120cbf9_fk_auth_group_id FOREIGN KEY (group_id) REFERENCES public.auth_group(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: auth_permission auth_permiss_content_type_id_2f476e4b_fk_django_content_type_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.auth_permission
		ADD CONSTRAINT auth_permiss_content_type_id_2f476e4b_fk_django_content_type_id FOREIGN KEY (content_type_id) REFERENCES public.django_content_type(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: auth_user_groups auth_user_groups_group_id_97559544_fk_auth_group_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.auth_user_groups
		ADD CONSTRAINT auth_user_groups_group_id_97559544_fk_auth_group_id FOREIGN KEY (group_id) REFERENCES public.auth_group(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: auth_user_groups auth_user_groups_user_id_6a12ed8b_fk_auth_user_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.auth_user_groups
		ADD CONSTRAINT auth_user_groups_user_id_6a12ed8b_fk_auth_user_id FOREIGN KEY (user_id) REFERENCES public.auth_user(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: auth_user_user_permissions auth_user_user_per_permission_id_1fbb5f2c_fk_auth_permission_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.auth_user_user_permissions
		ADD CONSTRAINT auth_user_user_per_permission_id_1fbb5f2c_fk_auth_permission_id FOREIGN KEY (permission_id) REFERENCES public.auth_permission(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: auth_user_user_permissions auth_user_user_permissions_user_id_a95ead1b_fk_auth_user_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.auth_user_user_permissions
		ADD CONSTRAINT auth_user_user_permissions_user_id_a95ead1b_fk_auth_user_id FOREIGN KEY (user_id) REFERENCES public.auth_user(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: django_admin_log django_admin_content_type_id_c4bce8eb_fk_django_content_type_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.django_admin_log
		ADD CONSTRAINT django_admin_content_type_id_c4bce8eb_fk_django_content_type_id FOREIGN KEY (content_type_id) REFERENCES public.django_content_type(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: django_admin_log django_admin_log_user_id_c564eba6_fk_auth_user_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.django_admin_log
		ADD CONSTRAINT django_admin_log_user_id_c564eba6_fk_auth_user_id FOREIGN KEY (user_id) REFERENCES public.auth_user(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_passwordresetrequest hand_confirmation_token_id_c5a214d4_fk_handmade_onetimetoken_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_passwordresetrequest
		ADD CONSTRAINT hand_confirmation_token_id_c5a214d4_fk_handmade_onetimetoken_id FOREIGN KEY (confirmation_token_id) REFERENCES public.handmade_onetimetoken(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_communicationchoice handm_option_id_72e3b574_fk_handmade_communicationchoicelist_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_communicationchoice
		ADD CONSTRAINT handm_option_id_72e3b574_fk_handmade_communicationchoicelist_id FOREIGN KEY (option_id) REFERENCES public.handmade_communicationchoicelist(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_userpending handma_activation_token_id_0b4a4b06_fk_handmade_onetimetoken_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_userpending
		ADD CONSTRAINT handma_activation_token_id_0b4a4b06_fk_handmade_onetimetoken_id FOREIGN KEY (activation_token_id) REFERENCES public.handmade_onetimetoken(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_memberextended_links handma_memberextended_id_366a0439_fk_handmade_memberextended_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_memberextended_links
		ADD CONSTRAINT handma_memberextended_id_366a0439_fk_handmade_memberextended_id FOREIGN KEY (memberextended_id) REFERENCES public.handmade_memberextended(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_asset handmade_asset_uploader_id_fc1cb702_fk_auth_user_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_asset
		ADD CONSTRAINT handmade_asset_uploader_id_fc1cb702_fk_auth_user_id FOREIGN KEY (uploader_id) REFERENCES public.auth_user(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_categorylastreadinfo handmade_category_member_id_c654be1e_fk_handmade_member_user_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_categorylastreadinfo
		ADD CONSTRAINT handmade_category_member_id_c654be1e_fk_handmade_member_user_id FOREIGN KEY (member_id) REFERENCES public.handmade_member(user_id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_category handmade_category_parent_id_ea91058d_fk_handmade_category_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_category
		ADD CONSTRAINT handmade_category_parent_id_ea91058d_fk_handmade_category_id FOREIGN KEY (parent_id) REFERENCES public.handmade_category(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_category handmade_category_project_id_54c6ecdb_fk_handmade_project_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_category
		ADD CONSTRAINT handmade_category_project_id_54c6ecdb_fk_handmade_project_id FOREIGN KEY (project_id) REFERENCES public.handmade_project(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_categorylastreadinfo handmade_categorylas_category_id_6248e1ce_fk_handmade_; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_categorylastreadinfo
		ADD CONSTRAINT handmade_categorylas_category_id_6248e1ce_fk_handmade_ FOREIGN KEY (category_id) REFERENCES public.handmade_category(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_communicationsubthread handmade_communic_member_id_5f2d479a_fk_handmade_member_user_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_communicationsubthread
		ADD CONSTRAINT handmade_communic_member_id_5f2d479a_fk_handmade_member_user_id FOREIGN KEY (member_id) REFERENCES public.handmade_member(user_id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_communicationchoice handmade_communic_member_id_6cbf0897_fk_handmade_member_user_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_communicationchoice
		ADD CONSTRAINT handmade_communic_member_id_6cbf0897_fk_handmade_member_user_id FOREIGN KEY (member_id) REFERENCES public.handmade_member(user_id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_communicationsubcategory handmade_communic_member_id_b5f5ebca_fk_handmade_member_user_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_communicationsubcategory
		ADD CONSTRAINT handmade_communic_member_id_b5f5ebca_fk_handmade_member_user_id FOREIGN KEY (member_id) REFERENCES public.handmade_member(user_id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_communicationsubcategory handmade_communica_category_id_0484fe33_fk_handmade_category_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_communicationsubcategory
		ADD CONSTRAINT handmade_communica_category_id_0484fe33_fk_handmade_category_id FOREIGN KEY (category_id) REFERENCES public.handmade_category(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_communicationchoicelist handmade_communicati_project_id_968e146e_fk_handmade_; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_communicationchoicelist
		ADD CONSTRAINT handmade_communicati_project_id_968e146e_fk_handmade_ FOREIGN KEY (project_id) REFERENCES public.handmade_project(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_communicationsubthread handmade_communication_thread_id_f3d65acd_fk_handmade_thread_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_communicationsubthread
		ADD CONSTRAINT handmade_communication_thread_id_f3d65acd_fk_handmade_thread_id FOREIGN KEY (thread_id) REFERENCES public.handmade_thread(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_discord handmade_discord_member_id_1c84599f_fk_handmade_member_user_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_discord
		ADD CONSTRAINT handmade_discord_member_id_1c84599f_fk_handmade_member_user_id FOREIGN KEY (member_id) REFERENCES public.handmade_member(user_id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_discordmessageattachment handmade_discordmess_asset_id_c64a3c31_fk_handmade_; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_discordmessageattachment
		ADD CONSTRAINT handmade_discordmess_asset_id_c64a3c31_fk_handmade_ FOREIGN KEY (asset_id) REFERENCES public.handmade_asset(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_discordmessagecontent handmade_discordmess_discord_id_1acc147f_fk_handmade_; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_discordmessagecontent
		ADD CONSTRAINT handmade_discordmess_discord_id_1acc147f_fk_handmade_ FOREIGN KEY (discord_id) REFERENCES public.handmade_discord(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_discordmessageembed handmade_discordmess_image_id_9b04bb5f_fk_handmade_; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_discordmessageembed
		ADD CONSTRAINT handmade_discordmess_image_id_9b04bb5f_fk_handmade_ FOREIGN KEY (image_id) REFERENCES public.handmade_asset(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_discordmessageembed handmade_discordmess_message_id_04f15ce6_fk_handmade_; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_discordmessageembed
		ADD CONSTRAINT handmade_discordmess_message_id_04f15ce6_fk_handmade_ FOREIGN KEY (message_id) REFERENCES public.handmade_discordmessage(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_discordmessagecontent handmade_discordmess_message_id_4dfde67d_fk_handmade_; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_discordmessagecontent
		ADD CONSTRAINT handmade_discordmess_message_id_4dfde67d_fk_handmade_ FOREIGN KEY (message_id) REFERENCES public.handmade_discordmessage(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_discordmessageattachment handmade_discordmess_message_id_d39da9b3_fk_handmade_; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_discordmessageattachment
		ADD CONSTRAINT handmade_discordmess_message_id_d39da9b3_fk_handmade_ FOREIGN KEY (message_id) REFERENCES public.handmade_discordmessage(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_discordmessageembed handmade_discordmess_video_id_1c41289f_fk_handmade_; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_discordmessageembed
		ADD CONSTRAINT handmade_discordmess_video_id_1c41289f_fk_handmade_ FOREIGN KEY (video_id) REFERENCES public.handmade_asset(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_kunenapost handmade_kunenapost_post_id_75b4faad_fk_handmade_post_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_kunenapost
		ADD CONSTRAINT handmade_kunenapost_post_id_75b4faad_fk_handmade_post_id FOREIGN KEY (post_id) REFERENCES public.handmade_post(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_kunenathread handmade_kunenathread_thread_id_fa6ae399_fk_handmade_thread_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_kunenathread
		ADD CONSTRAINT handmade_kunenathread_thread_id_fa6ae399_fk_handmade_thread_id FOREIGN KEY (thread_id) REFERENCES public.handmade_thread(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_libraryresource handmade_libraryreso_category_id_06d45134_fk_handmade_; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_libraryresource
		ADD CONSTRAINT handmade_libraryreso_category_id_06d45134_fk_handmade_ FOREIGN KEY (category_id) REFERENCES public.handmade_category(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_libraryresource_media_types handmade_libraryreso_librarymediatype_id_9589f3f4_fk_handmade_; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_libraryresource_media_types
		ADD CONSTRAINT handmade_libraryreso_librarymediatype_id_9589f3f4_fk_handmade_ FOREIGN KEY (librarymediatype_id) REFERENCES public.handmade_librarymediatype(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_libraryresource_topics handmade_libraryreso_libraryresource_id_74e35c80_fk_handmade_; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_libraryresource_topics
		ADD CONSTRAINT handmade_libraryreso_libraryresource_id_74e35c80_fk_handmade_ FOREIGN KEY (libraryresource_id) REFERENCES public.handmade_libraryresource(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_libraryresource_media_types handmade_libraryreso_libraryresource_id_f643f15f_fk_handmade_; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_libraryresource_media_types
		ADD CONSTRAINT handmade_libraryreso_libraryresource_id_f643f15f_fk_handmade_ FOREIGN KEY (libraryresource_id) REFERENCES public.handmade_libraryresource(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_libraryresource_topics handmade_libraryreso_librarytopic_id_7aa70be0_fk_handmade_; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_libraryresource_topics
		ADD CONSTRAINT handmade_libraryreso_librarytopic_id_7aa70be0_fk_handmade_ FOREIGN KEY (librarytopic_id) REFERENCES public.handmade_librarytopic(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_libraryresource handmade_libraryreso_project_id_a3d6e0fe_fk_handmade_; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_libraryresource
		ADD CONSTRAINT handmade_libraryreso_project_id_a3d6e0fe_fk_handmade_ FOREIGN KEY (project_id) REFERENCES public.handmade_project(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_libraryresourcestar handmade_libraryreso_resource_id_5db93928_fk_handmade_; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_libraryresourcestar
		ADD CONSTRAINT handmade_libraryreso_resource_id_5db93928_fk_handmade_ FOREIGN KEY (resource_id) REFERENCES public.handmade_libraryresource(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_libraryresourcestar handmade_libraryresourcestar_user_id_b8483e28_fk_auth_user_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_libraryresourcestar
		ADD CONSTRAINT handmade_libraryresourcestar_user_id_b8483e28_fk_auth_user_id FOREIGN KEY (user_id) REFERENCES public.auth_user(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_librarytopic handmade_librarytopi_parent_id_5dfddf8e_fk_handmade_; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_librarytopic
		ADD CONSTRAINT handmade_librarytopi_parent_id_5dfddf8e_fk_handmade_ FOREIGN KEY (parent_id) REFERENCES public.handmade_librarytopic(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_librarytopic handmade_librarytopi_project_id_3b1879da_fk_handmade_; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_librarytopic
		ADD CONSTRAINT handmade_librarytopi_project_id_3b1879da_fk_handmade_ FOREIGN KEY (project_id) REFERENCES public.handmade_project(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_license_texts handmade_license_t_license_id_93d0ac5d_fk_handmade_license_slug; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_license_texts
		ADD CONSTRAINT handmade_license_t_license_id_93d0ac5d_fk_handmade_license_slug FOREIGN KEY (license_id) REFERENCES public.handmade_license(slug) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_license_texts handmade_license_texts_post_id_a7dc9630_fk_handmade_post_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_license_texts
		ADD CONSTRAINT handmade_license_texts_post_id_a7dc9630_fk_handmade_post_id FOREIGN KEY (post_id) REFERENCES public.handmade_post(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_member handmade_mem_extended_id_8e656d93_fk_handmade_memberextended_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_member
		ADD CONSTRAINT handmade_mem_extended_id_8e656d93_fk_handmade_memberextended_id FOREIGN KEY (extended_id) REFERENCES public.handmade_memberextended(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_member_projects handmade_member_p_member_id_70115602_fk_handmade_member_user_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_member_projects
		ADD CONSTRAINT handmade_member_p_member_id_70115602_fk_handmade_member_user_id FOREIGN KEY (member_id) REFERENCES public.handmade_member(user_id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_member_projects handmade_member_proj_project_id_8b14279f_fk_handmade_project_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_member_projects
		ADD CONSTRAINT handmade_member_proj_project_id_8b14279f_fk_handmade_project_id FOREIGN KEY (project_id) REFERENCES public.handmade_project(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_member handmade_member_user_id_9ee4a7ad_fk_auth_user_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_member
		ADD CONSTRAINT handmade_member_user_id_9ee4a7ad_fk_auth_user_id FOREIGN KEY (user_id) REFERENCES public.auth_user(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_memberextended_links handmade_memberextended__links_id_9161abc0_fk_handmade_links_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_memberextended_links
		ADD CONSTRAINT handmade_memberextended__links_id_9161abc0_fk_handmade_links_id FOREIGN KEY (links_id) REFERENCES public.handmade_links(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_project_languages handmade_p_codelanguage_id_55ed9fee_fk_handmade_codelanguage_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_project_languages
		ADD CONSTRAINT handmade_p_codelanguage_id_55ed9fee_fk_handmade_codelanguage_id FOREIGN KEY (codelanguage_id) REFERENCES public.handmade_codelanguage(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_passwordresetrequest handmade_passwordresetrequest_user_id_8cbcaa87_fk_auth_user_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_passwordresetrequest
		ADD CONSTRAINT handmade_passwordresetrequest_user_id_8cbcaa87_fk_auth_user_id FOREIGN KEY (user_id) REFERENCES public.auth_user(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_podcast handmade_podcast_image_id_cfbd1a68_fk_handmade_imagefile_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_podcast
		ADD CONSTRAINT handmade_podcast_image_id_cfbd1a68_fk_handmade_imagefile_id FOREIGN KEY (image_id) REFERENCES public.handmade_imagefile(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_podcast handmade_podcast_project_id_bf27fb3a_fk_handmade_project_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_podcast
		ADD CONSTRAINT handmade_podcast_project_id_bf27fb3a_fk_handmade_project_id FOREIGN KEY (project_id) REFERENCES public.handmade_project(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_podcastepisode handmade_podcastepis_podcast_id_b86d4941_fk_handmade_; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_podcastepisode
		ADD CONSTRAINT handmade_podcastepis_podcast_id_b86d4941_fk_handmade_ FOREIGN KEY (podcast_id) REFERENCES public.handmade_podcast(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_post handmade_post_author_id_f056f9c0_fk_handmade_member_user_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_post
		ADD CONSTRAINT handmade_post_author_id_f056f9c0_fk_handmade_member_user_id FOREIGN KEY (author_id) REFERENCES public.handmade_member(user_id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_post handmade_post_category_id_051797a3_fk_handmade_category_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_post
		ADD CONSTRAINT handmade_post_category_id_051797a3_fk_handmade_category_id FOREIGN KEY (category_id) REFERENCES public.handmade_category(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_post handmade_post_current_id_762211b7_fk_handmade_; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_post
		ADD CONSTRAINT handmade_post_current_id_762211b7_fk_handmade_ FOREIGN KEY (current_id) REFERENCES public.handmade_posttextversion(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_post handmade_post_parent_id_2a784009_fk_handmade_post_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_post
		ADD CONSTRAINT handmade_post_parent_id_2a784009_fk_handmade_post_id FOREIGN KEY (parent_id) REFERENCES public.handmade_post(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_post handmade_post_thread_id_96319481_fk_handmade_thread_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_post
		ADD CONSTRAINT handmade_post_thread_id_96319481_fk_handmade_thread_id FOREIGN KEY (thread_id) REFERENCES public.handmade_thread(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_posttextversion handmade_posttextver_editor_id_62fdd463_fk_handmade_; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_posttextversion
		ADD CONSTRAINT handmade_posttextver_editor_id_62fdd463_fk_handmade_ FOREIGN KEY (editor_id) REFERENCES public.handmade_member(user_id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_posttextversion handmade_posttextver_text_id_4e0fde60_fk_handmade_; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_posttextversion
		ADD CONSTRAINT handmade_posttextver_text_id_4e0fde60_fk_handmade_ FOREIGN KEY (text_id) REFERENCES public.handmade_posttext(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_posttextversion handmade_posttextversion_post_id_440a419c_fk_handmade_post_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_posttextversion
		ADD CONSTRAINT handmade_posttextversion_post_id_440a419c_fk_handmade_post_id FOREIGN KEY (post_id) REFERENCES public.handmade_post(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_project handmade_project_annotation_id_e8b62fac_fk_handmade_category_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_project
		ADD CONSTRAINT handmade_project_annotation_id_e8b62fac_fk_handmade_category_id FOREIGN KEY (annotation_id) REFERENCES public.handmade_category(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_project handmade_project_blog_id_a1edc139_fk_handmade_category_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_project
		ADD CONSTRAINT handmade_project_blog_id_a1edc139_fk_handmade_category_id FOREIGN KEY (blog_id) REFERENCES public.handmade_category(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_project_downloads handmade_project_dow_project_id_3b65cea6_fk_handmade_project_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_project_downloads
		ADD CONSTRAINT handmade_project_dow_project_id_3b65cea6_fk_handmade_project_id FOREIGN KEY (project_id) REFERENCES public.handmade_project(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_project handmade_project_forum_id_1a2c50dc_fk_handmade_category_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_project
		ADD CONSTRAINT handmade_project_forum_id_1a2c50dc_fk_handmade_category_id FOREIGN KEY (forum_id) REFERENCES public.handmade_category(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_project_groups handmade_project_gro_project_id_e5d19819_fk_handmade_project_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_project_groups
		ADD CONSTRAINT handmade_project_gro_project_id_e5d19819_fk_handmade_project_id FOREIGN KEY (project_id) REFERENCES public.handmade_project(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_project_groups handmade_project_groups_group_id_473e9ef3_fk_auth_group_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_project_groups
		ADD CONSTRAINT handmade_project_groups_group_id_473e9ef3_fk_auth_group_id FOREIGN KEY (group_id) REFERENCES public.auth_group(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_project_screenshots handmade_project_imagefile_id_20b22b64_fk_handmade_imagefile_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_project_screenshots
		ADD CONSTRAINT handmade_project_imagefile_id_20b22b64_fk_handmade_imagefile_id FOREIGN KEY (imagefile_id) REFERENCES public.handmade_imagefile(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_project_licenses handmade_project_l_license_id_618488c2_fk_handmade_license_slug; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_project_licenses
		ADD CONSTRAINT handmade_project_l_license_id_618488c2_fk_handmade_license_slug FOREIGN KEY (license_id) REFERENCES public.handmade_license(slug) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_project_languages handmade_project_lan_project_id_43c828c0_fk_handmade_project_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_project_languages
		ADD CONSTRAINT handmade_project_lan_project_id_43c828c0_fk_handmade_project_id FOREIGN KEY (project_id) REFERENCES public.handmade_project(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_project_licenses handmade_project_lic_project_id_83ea8a77_fk_handmade_project_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_project_licenses
		ADD CONSTRAINT handmade_project_lic_project_id_83ea8a77_fk_handmade_project_id FOREIGN KEY (project_id) REFERENCES public.handmade_project(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_project_links handmade_project_lin_project_id_fa326174_fk_handmade_project_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_project_links
		ADD CONSTRAINT handmade_project_lin_project_id_fa326174_fk_handmade_project_id FOREIGN KEY (project_id) REFERENCES public.handmade_project(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_project_links handmade_project_links_links_id_ffe7237e_fk_handmade_links_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_project_links
		ADD CONSTRAINT handmade_project_links_links_id_ffe7237e_fk_handmade_links_id FOREIGN KEY (links_id) REFERENCES public.handmade_links(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_project_downloads handmade_project_otherfile_id_9aa4b8a7_fk_handmade_otherfile_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_project_downloads
		ADD CONSTRAINT handmade_project_otherfile_id_9aa4b8a7_fk_handmade_otherfile_id FOREIGN KEY (otherfile_id) REFERENCES public.handmade_otherfile(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_project handmade_project_parent_id_fa896aae_fk_handmade_project_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_project
		ADD CONSTRAINT handmade_project_parent_id_fa896aae_fk_handmade_project_id FOREIGN KEY (parent_id) REFERENCES public.handmade_project(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_project_screenshots handmade_project_scr_project_id_68ab5766_fk_handmade_project_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_project_screenshots
		ADD CONSTRAINT handmade_project_scr_project_id_68ab5766_fk_handmade_project_id FOREIGN KEY (project_id) REFERENCES public.handmade_project(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_project handmade_project_static_id_7473d585_fk_handmade_category_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_project
		ADD CONSTRAINT handmade_project_static_id_7473d585_fk_handmade_category_id FOREIGN KEY (static_id) REFERENCES public.handmade_category(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_project handmade_project_wiki_id_eba06ae0_fk_handmade_category_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_project
		ADD CONSTRAINT handmade_project_wiki_id_eba06ae0_fk_handmade_category_id FOREIGN KEY (wiki_id) REFERENCES public.handmade_category(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_snippet handmade_snippet_asset_id_c786de4f_fk_handmade_asset_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_snippet
		ADD CONSTRAINT handmade_snippet_asset_id_c786de4f_fk_handmade_asset_id FOREIGN KEY (asset_id) REFERENCES public.handmade_asset(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_snippet handmade_snippet_discord_message_id_d16f1f4e_fk_handmade_; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_snippet
		ADD CONSTRAINT handmade_snippet_discord_message_id_d16f1f4e_fk_handmade_ FOREIGN KEY (discord_message_id) REFERENCES public.handmade_discordmessage(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_snippet handmade_snippet_owner_id_fcca1783_fk_auth_user_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_snippet
		ADD CONSTRAINT handmade_snippet_owner_id_fcca1783_fk_auth_user_id FOREIGN KEY (owner_id) REFERENCES public.auth_user(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_thread handmade_thread_category_id_425353c3_fk_handmade_category_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_thread
		ADD CONSTRAINT handmade_thread_category_id_425353c3_fk_handmade_category_id FOREIGN KEY (category_id) REFERENCES public.handmade_category(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_thread handmade_thread_first_id_c3bcaf2f_fk_handmade_post_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_thread
		ADD CONSTRAINT handmade_thread_first_id_c3bcaf2f_fk_handmade_post_id FOREIGN KEY (first_id) REFERENCES public.handmade_post(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_thread handmade_thread_last_id_dcb893b6_fk_handmade_post_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_thread
		ADD CONSTRAINT handmade_thread_last_id_dcb893b6_fk_handmade_post_id FOREIGN KEY (last_id) REFERENCES public.handmade_post(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_threadlastreadinfo handmade_threadla_member_id_3d92b34f_fk_handmade_member_user_id; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_threadlastreadinfo
		ADD CONSTRAINT handmade_threadla_member_id_3d92b34f_fk_handmade_member_user_id FOREIGN KEY (member_id) REFERENCES public.handmade_member(user_id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_threadlastreadinfo handmade_threadlastr_category_id_d55eedeb_fk_handmade_; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_threadlastreadinfo
		ADD CONSTRAINT handmade_threadlastr_category_id_d55eedeb_fk_handmade_ FOREIGN KEY (category_id) REFERENCES public.handmade_category(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- Name: handmade_threadlastreadinfo handmade_threadlastr_thread_id_783da622_fk_handmade_; Type: FK CONSTRAINT; Schema: public; Owner: hmn
	--
	
	ALTER TABLE ONLY public.handmade_threadlastreadinfo
		ADD CONSTRAINT handmade_threadlastr_thread_id_783da622_fk_handmade_ FOREIGN KEY (thread_id) REFERENCES public.handmade_thread(id) DEFERRABLE INITIALLY DEFERRED;
	
	
	--
	-- PostgreSQL database dump complete
	--
	
	
	`)
	if err != nil {
		return err
	}

	return nil
}

func (m Initial) Down(tx pgx.Tx) error {
	panic("nope, ha ha, I'm the initial migration. how did you even run this function anyway")
}
