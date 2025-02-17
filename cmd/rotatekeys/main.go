package main

import (
	"os"

	"github.com/rs/zerolog"

	"github.com/a-novel-kit/context"

	"github.com/a-novel/authentication/config"
	"github.com/a-novel/authentication/internal/dao"
	"github.com/a-novel/authentication/internal/lib"
	"github.com/a-novel/authentication/internal/services"
	"github.com/a-novel/authentication/models"
)

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

	searchKeysDAO := dao.NewSearchKeysRepository()
	insertKeyDAO := dao.NewInsertKeyRepository()

	generateKeysService := services.NewGenerateKeyService(
		services.NewGenerateKeySource(searchKeysDAO, insertKeyDAO),
	)

	var countGeneratedKeys, countFailedKeys int

	for _, usage := range models.KnownKeyUsages {
		keyID, err := generateKeysService.GenerateKey(ctx, usage)
		if err != nil {
			logger.Error().Err(err).Str("usage", string(usage)).Msg("generate keys")

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
