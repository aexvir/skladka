package config

import (
	"fmt"

	"github.com/ardanlabs/conf/v3"

	"github.com/aexvir/skladka/internal/errors"
)

var (
	BuildBranch   = "local"
	BuildRevision = "dev"
	BuildDate     = "2006-01-02T15:04:05Z07:00"
)

// ErrHelpWanted indicates that the --help flag was passed to the binary
var ErrHelpWanted = errors.New("help requested")

type Config struct {
	conf.Version

	Core
	Postgres
	Observability
}

type Core struct {
	// Current deployment hostname.
	Hostname string `conf:"hostname,env:HOSTNAME"`
	// EncryptionKey used to encrypt all paste data.
	EncryptionKey string `conf:"encryption-key,env:ENCRYPTION_KEY"`
	// Environment the application is running in.
	Environment string `conf:"env,env:ENVIRONMENT,default:dev"`
}

type Postgres struct {
	// Host where the postgres db can be reached at.
	Host string `conf:"host,env:POSTGRES_HOST"`
	// Port where the database is listening in.
	Port int `conf:"port,env:POSTGRES_PORT,default:5432"`
	// User to use for database queries.
	User string `conf:"user,env:POSTGRES_USER"`
	// Password for the db user.
	Password string `conf:"pass,env:POSTGRES_PASS"`
	// Database the application should connect to.
	Database string `conf:"db,env:POSTGRES_DB"`
	// URL is the full postgres connection URL in the format: postgres://user:pass@host:port/db
	URL string `conf:"url,env:POSTGRES_DB_URL"`
}

type Observability struct {
	Logging Otlp `conf:"logging"`
	Metrics Otlp `conf:"metrics"`
	Tracing Otlp `conf:"tracing"`
}

type Otlp struct {
	// Enabled controls if this otlp exporter should be used or not.
	Enabled bool `conf:"enabled"`
	// Host where the otlp collector can be reached at.
	Host string `conf:"host"`
	// Port where the otlp collector is listening in.
	Port int `conf:"port"`
}

func Load() (Config, error) {
	var cfg Config

	// set version information
	cfg.Version.Build = BuildRevision
	cfg.Version.Desc = fmt.Sprintf("Branch: %s, Build Date: %s", BuildBranch, BuildDate)

	help, err := conf.Parse("SKD", &cfg)
	if err != nil {
		if errors.Is(err, conf.ErrHelpWanted) {
			return cfg, &helperr{help: help, err: ErrHelpWanted}
		}
		return cfg, errors.Wrap(err, "parsing config")
	}

	return cfg, nil
}

type helperr struct {
	err  error
	help string
}

func (e *helperr) Error() string {
	return e.help
}

func (e *helperr) Unwrap() error {
	return e.err
}
