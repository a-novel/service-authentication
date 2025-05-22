package main

import (
	"context"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/rs/zerolog"

	"github.com/a-novel/service-authentication/config"
	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/lib"
	"github.com/a-novel/service-authentication/internal/services"
	"github.com/a-novel/service-authentication/models"
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
		sentryOptions := sentry.ClientOptions{Dsn: config.Sentry.DSN}

		if err = sentry.Init(sentryOptions); err != nil {
			logger.Fatal().Err(err).Msg("initialize sentry")
		}

		defer sentry.Flush(SentryFlushTimeout)
		ctx = sentry.SetHubOnContext(ctx, sentry.CurrentHub())
	}

	hub := sentry.GetHubFromContext(ctx)
	if hub == nil {
		hub = sentry.CurrentHub().Clone()
	}

	searchKeysDAO := dao.NewSearchKeysRepository()
	insertKeyDAO := dao.NewInsertKeyRepository()

	generateKeysService := services.NewGenerateKeyService(
		services.NewGenerateKeySource(searchKeysDAO, insertKeyDAO),
	)

	var countGeneratedKeys, countFailedKeys int

	for _, usage := range models.KnownKeyUsages {
		hub.AddBreadcrumb(&sentry.Breadcrumb{
			Message:  "generate key",
			Category: "generate key",
			Data:     map[string]any{"usage": usage},
		}, nil)

		keyID, err := generateKeysService.GenerateKey(ctx, usage)
		if err != nil {
			logger.Error().Err(err).Str("usage", string(usage)).Msg("generate keys")
			hub.CaptureException(err)

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
