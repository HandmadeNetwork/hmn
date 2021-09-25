package config

import (
	"fmt"

	"github.com/jackc/pgx/v4"
	"github.com/rs/zerolog"
)

type Environment string

const (
	Live Environment = "live"
	Beta             = "beta"
	Dev              = "dev"
)

type HMNConfig struct {
	Env          Environment
	Addr         string
	PrivateAddr  string
	BaseUrl      string
	LogLevel     zerolog.Level
	Postgres     PostgresConfig
	Auth         AuthConfig
	Admin        AdminConfig
	Email        EmailConfig
	DigitalOcean DigitalOceanConfig
	Discord      DiscordConfig
	EpisodeGuide EpisodeGuide
}

type PostgresConfig struct {
	User     string
	Password string
	Hostname string
	Port     int
	DbName   string
	LogLevel pgx.LogLevel
	MinConn  int32
	MaxConn  int32
}

type AuthConfig struct {
	CookieDomain string
	CookieSecure bool
}

type DigitalOceanConfig struct {
	AssetsSpacesKey      string
	AssetsSpacesSecret   string
	AssetsSpacesRegion   string
	AssetsSpacesEndpoint string
	AssetsSpacesBucket   string
	AssetsPathPrefix     string
	AssetsPublicUrlRoot  string
}

type EmailConfig struct {
	ServerAddress          string
	ServerPort             int
	FromAddress            string
	FromAddressPassword    string
	FromName               string
	OverrideRecipientEmail string
}

type DiscordConfig struct {
	BotToken  string
	BotUserID string

	OAuthClientID     string
	OAuthClientSecret string

	GuildID              string
	MemberRoleID         string
	ShowcaseChannelID    string
	LibraryChannelID     string
	JamShowcaseChannelID string
}

type EpisodeGuide struct {
	CineraOutputPath string
	Projects         map[string]string // NOTE(asaf): Maps from slugs to default episode guide topic
}

type AdminConfig struct {
	AtomUsername string
	AtomPassword string
}

func init() {
	if Config.EpisodeGuide.Projects == nil {
		Config.EpisodeGuide.Projects = make(map[string]string)
	}
}

func (info PostgresConfig) DSN() string {
	return fmt.Sprintf("user=%s password=%s host=%s port=%d dbname=%s", info.User, info.Password, info.Hostname, info.Port, info.DbName)
}
