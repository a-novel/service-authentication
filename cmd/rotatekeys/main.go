package main

import (
	"net/http"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/rs/zerolog"

	"github.com/a-novel-kit/context"
	sentryctx "github.com/a-novel-kit/context/sentry"

	"github.com/a-novel/authentication/config"
	"github.com/a-novel/authentication/internal/dao"
	"github.com/a-novel/authentication/internal/lib"
	"github.com/a-novel/authentication/internal/services"
	"github.com/a-novel/authentication/models"
)

const SentryFlushTimeout = 2 * time.Second

func main() {
	logger := zerolog.New(os.Stdout).With().
		Str("app", "authentication").
		Str("job", "rotate-keys").
		Timestamp().
		Logger()

	if config.LoggerColor {
		logger = logger.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	ctx, err := lib.NewAgoraContext(context.Background())
	if err != nil {
		logger.Fatal().Err(err).Msg("initialize agora context")
	}

	if config.Sentry.DSN != "" {
		sentryOptions := sentry.ClientOptions{
			Dsn: config.Sentry.DSN,
			BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
				if req, ok := hint.Context.Value(sentry.RequestContextKey).(*http.Request); ok {
					// Add IP Address to user information.
					event.User.IPAddress = req.RemoteAddr
				}

				return event
			},
		}

		if err = sentry.Init(sentryOptions); err != nil {
			logger.Fatal().Err(err).Msg("initialize sentry")
		}

		defer sentry.Flush(SentryFlushTimeout)

		ctx = sentry.SetHubOnContext(ctx, sentry.CurrentHub())
	}

	searchKeysDAO := dao.NewSearchKeysRepository()
	insertKeyDAO := dao.NewInsertKeyRepository()

	generateKeysService := services.NewGenerateKeyService(
		services.NewGenerateKeySource(searchKeysDAO, insertKeyDAO),
	)

	var countGeneratedKeys, countFailedKeys int

	for _, usage := range models.KnownKeyUsages {
		sentryctx.AddBreadcrumb(ctx, &sentry.Breadcrumb{
			Message:  "generate key",
			Category: "generate key",
			Data:     map[string]any{"usage": usage},
		}, nil)

		keyID, err := generateKeysService.GenerateKey(ctx, usage)
		if err != nil {
			logger.Error().Err(err).Str("usage", string(usage)).Msg("generate keys")
			sentryctx.CaptureException(ctx, err)

			countFailedKeys++

			continue
		}

		if keyID != nil {
			countGeneratedKeys++

			logger.Info().Str("usage", string(usage)).Str("key_id", keyID.String()).Msg("key generated")

			continue
		}

		logger.Info().Str("usage", string(usage)).Msg("no key generated")
	}

	logger.Info().
		Int("total_keys", len(models.KnownKeyUsages)).
		Int("generated_keys", countGeneratedKeys).
		Int("failed_keys", countFailedKeys).
		Msg("rotation done")
}
