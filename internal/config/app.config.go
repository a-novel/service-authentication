package config

import (
	"time"

	"github.com/a-novel-kit/golib/grpcf"
	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
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
	ServiceJsonKeysHost        string                    `json:"jsonKeysServiceHost" yaml:"jsonKeysServiceHost"`
	ServiceJsonKeysPort        int                       `json:"jsonKeysServicePort" yaml:"jsonKeysServicePort"`
	ServiceJsonKeysCredentials grpcf.CredentialsProvider `json:"-"                   yaml:"-"`
}

// Limits caps what one caller, one mailbox or a burst can make the REST API spend. Rates are
// written "<requests>/<window>" in their variables.
type Limits struct {
	// Strict caps each caller of a route issuing anonymous tokens or invitation requests.
	Strict httpf.Rate `json:"strict" yaml:"strict"`
	// LoginBurst caps each visitor's logins over a short window.
	LoginBurst httpf.Rate `json:"loginBurst" yaml:"loginBurst"`
	// Login caps each visitor's logins, and the logins naming one email, over a longer window.
	Login httpf.Rate `json:"login" yaml:"login"`
	// Mailbox caps the emails each address receives from one route.
	Mailbox httpf.Rate `json:"mailbox" yaml:"mailbox"`
	// WaitlistMaxConcurrent caps the waitlist requests waiting on Google at once, per instance.
	WaitlistMaxConcurrent int `json:"waitlistMaxConcurrent" yaml:"waitlistMaxConcurrent"`
	// WaitlistMaxRequestSize caps a waitlist request body, in bytes.
	WaitlistMaxRequestSize int64 `json:"waitlistMaxRequestSize" yaml:"waitlistMaxRequestSize"`
	// Argon2MaxConcurrent caps the password hashes running at once, per instance.
	Argon2MaxConcurrent int `json:"argon2MaxConcurrent" yaml:"argon2MaxConcurrent"`
}

// App is the fully resolved configuration for the service, assembled at startup from the environment.
type App struct {
	App    Main               `json:"app"    yaml:"app"`
	Rest   httpf.ServerConfig `json:"rest"   yaml:"rest"`
	Limits Limits             `json:"limits" yaml:"limits"`

	DependenciesConfig Dependencies        `json:"dependencies" yaml:"dependencies"`
	Permissions        Permissions         `json:"permissions"  yaml:"permissions"`
	ShortCodesConfig   ShortCodes          `json:"shortCodes"   yaml:"shortCodes"`
	SmtpUrlsConfig     SmtpUrls            `json:"smtpUrls"     yaml:"smtpUrls"`
	Waitlist           authconfig.Waitlist `json:"waitlist"     yaml:"waitlist"`

	Smtp       smtp.Sender        `json:"smtp"       yaml:"smtp"`
	Otel       otel.Config        `json:"otel"       yaml:"otel"`
	Logger     logging.Log        `json:"logger"     yaml:"logger"`
	HttpLogger logging.HTTPConfig `json:"httplogger" yaml:"httplogger"`
	Postgres   postgres.Config    `json:"postgres"   yaml:"postgres"`
}
