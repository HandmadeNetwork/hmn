package config

import (
	"fmt"

	"github.com/jackc/pgx/v5/tracelog"
	"github.com/rs/zerolog"
)

type Environment string

const (
	Live Environment = "live"
	Beta             = "beta"
	Dev              = "dev"
)

type HMNConfig struct {
	Env               Environment
	Addr              string
	PrivateAddr       string
	BaseUrl           string
	LogLevel          zerolog.Level
	Postgres          PostgresConfig
	Auth              AuthConfig
	Admin             AdminConfig
	Email             EmailConfig
	DigitalOcean      DigitalOceanConfig
	Discord           DiscordConfig
	Twitch            TwitchConfig
	EpisodeGuide      EpisodeGuide
	DevConfig         DevConfig
	PreviewGeneration PreviewGenerationConfig
	Calendars         []CalendarSource
	EsBuild           EsBuildConfig
	Postmark          PostmarkConfig
}

type PostgresConfig struct {
	User                 string
	Password             string
	Hostname             string
	Port                 int
	DbName               string
	LogLevel             tracelog.LogLevel
	MinConn              int32
	MaxConn              int32
	SlowQueryThresholdMs int
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
	AssetsPublicUrlRoot  string

	RunFakeServer bool
	FakeAddr      string
}

type EmailConfig struct {
	ServerAddress  string
	ServerPort     int
	FromAddress    string
	MailerUsername string
	MailerPassword string
	FromName       string
	ForceToAddress string
}

type DiscordConfig struct {
	BotToken  string
	BotUserID string

	OAuthClientID     string
	OAuthClientSecret string

	GuildID           string
	MemberRoleID      string
	ShowcaseChannelID string
	LibraryChannelID  string
	StreamsChannelID  string
}

type TwitchConfig struct {
	ClientID       string
	ClientSecret   string
	EventSubSecret string // NOTE(asaf): Between 10-100 chars long. Anything will do.
	BaseUrl        string
	BaseIDUrl      string
}

type CalendarSource struct {
	Name string
	Url  string
}

type EpisodeGuide struct {
	CineraOutputPath string
	Projects         map[string]string // NOTE(asaf): Maps from slugs to default episode guide topic
}

type AdminConfig struct {
	AtomUsername string
	AtomPassword string
}

type DevConfig struct {
	LiveTemplates bool // load templates live from the filesystem instead of embedding them
}

type PreviewGenerationConfig struct {
	FFMpegPath   string
	CPULimitPath string
}

type EsBuildConfig struct {
	Port uint16
}

type PostmarkConfig struct {
	TransactionalStreamToken string
}

func init() {
	if Config.EpisodeGuide.Projects == nil {
		Config.EpisodeGuide.Projects = make(map[string]string)
	}
}

func (info PostgresConfig) DSN() string {
	return fmt.Sprintf("user=%s password=%s host=%s port=%d dbname=%s", info.User, info.Password, info.Hostname, info.Port, info.DbName)
}
