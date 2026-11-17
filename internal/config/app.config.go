package config

import (
	"time"

	"github.com/a-novel-kit/golib/grpcf"
	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
	"github.com/a-novel-kit/golib/servicetoken"
	"github.com/a-novel-kit/golib/smtp"

	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
)

// Main holds the core identity of the service, used to tag its logs and traces.
type Main struct {
	Name string
	// DowntimeStart is when a planned downtime starts; nil when none is planned. From then until
	// it is removed, the service refuses work and leaves its database alone.
	DowntimeStart *time.Time
}

// Dependencies configures how the service reaches the backing services it calls.
type Dependencies struct {
	ServiceJSONKeys grpcf.ClientConfig
}

// App is the fully resolved configuration for the service, assembled at startup from the environment.
type App struct {
	App    Main
	Rest   httpf.ServerConfig
	Limits authconfig.Limits
	// ServiceToken names the service tokens the REST server accepts. Without an issuer, the server
	// refuses to start unless IssuerOptional is set.
	ServiceToken servicetoken.Config

	Dependencies Dependencies
	Permissions  authconfig.Permissions
	ShortCodes   authconfig.ShortCodes
	OAuth        authconfig.OAuth
	SMTPURLs     authconfig.SMTPURLs
	Waitlist     authconfig.Waitlist

	SMTP smtp.Sender
	// SMTPPing reaches the same relay within the health check's short budget.
	SMTPPing   smtp.Sender
	Otel       otel.Config
	Logger     logging.Log
	RestLogger logging.HTTPConfig
	Postgres   postgres.Config
}
