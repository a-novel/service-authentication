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
	Name string `json:"name" yaml:"name"`
	// DowntimeStart is when a planned downtime starts; nil when none is planned. From then until
	// it is removed, the service refuses work and leaves its database alone.
	DowntimeStart *time.Time `json:"downtimeStart" yaml:"downtimeStart"`
}

// Dependencies configures how the service reaches the backing services it calls.
type Dependencies struct {
	ServiceJSONKeysHost        string                    `json:"jsonKeysServiceHost" yaml:"jsonKeysServiceHost"`
	ServiceJSONKeysPort        int                       `json:"jsonKeysServicePort" yaml:"jsonKeysServicePort"`
	ServiceJSONKeysCredentials grpcf.CredentialsProvider `json:"-"                   yaml:"-"`
}

// App is the fully resolved configuration for the service, assembled at startup from the environment.
type App struct {
	App    Main               `json:"app"    yaml:"app"`
	Rest   httpf.ServerConfig `json:"rest"   yaml:"rest"`
	Limits authconfig.Limits  `json:"limits" yaml:"limits"`
	// ServiceToken names the service tokens the REST server accepts. Without an issuer, the server
	// refuses to start unless ServiceIssuerOptional is set.
	ServiceToken servicetoken.Config `json:"serviceToken" yaml:"serviceToken"`
	// ServiceIssuerOptional lets the REST server start without a service-token issuer, and then
	// take every request. Only a local or test stack sets it.
	ServiceIssuerOptional bool `json:"serviceIssuerOptional" yaml:"serviceIssuerOptional"`

	Dependencies Dependencies           `json:"dependencies" yaml:"dependencies"`
	Permissions  authconfig.Permissions `json:"permissions"  yaml:"permissions"`
	ShortCodes   authconfig.ShortCodes  `json:"shortCodes"   yaml:"shortCodes"`
	OAuth        authconfig.OAuth       `json:"oauth"        yaml:"oauth"`
	SMTPURLs     authconfig.SMTPURLs    `json:"smtpUrls"     yaml:"smtpUrls"`
	Waitlist     authconfig.Waitlist    `json:"waitlist"     yaml:"waitlist"`

	SMTP smtp.Sender `json:"smtp" yaml:"smtp"`
	// SMTPPing reaches the same relay within the health check's short budget.
	SMTPPing   smtp.Sender        `json:"smtpPing"   yaml:"smtpPing"`
	Otel       otel.Config        `json:"otel"       yaml:"otel"`
	Logger     logging.Log        `json:"logger"     yaml:"logger"`
	RestLogger logging.HTTPConfig `json:"restLogger" yaml:"restLogger"`
	Postgres   postgres.Config    `json:"postgres"   yaml:"postgres"`
}
