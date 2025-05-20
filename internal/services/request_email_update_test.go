package services_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/config"
	"github.com/a-novel/service-authentication/internal/services"
	servicesmocks "github.com/a-novel/service-authentication/internal/services/mocks"
	"github.com/a-novel/service-authentication/models"
)

func TestRequestEmailUpdate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type createShortCodeData struct {
		resp *models.ShortCode
		err  error
	}

	testCases := []struct {
		name string

		request services.RequestEmailUpdateRequest

		createShortCodeData *createShortCodeData

		expectErr error
	}{
		{
			name: "Success",

			request: services.RequestEmailUpdateRequest{
				Email: "user@provider.com",
				ID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},

			createShortCodeData: &createShortCodeData{
				resp: &models.ShortCode{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Usage:     "test-usage",
					Target:    "test-target",
					Data:      []byte("test-data"),
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					ExpiresAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
					PlainCode: "abcdef123456",
				},
			},
		},
		{
			name: "CreateShortCodeError",

			request: services.RequestEmailUpdateRequest{
				Email: "user@provider.com",
				ID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},

			createShortCodeData: &createShortCodeData{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			r, w, err := os.Pipe()
			require.NoError(t, err)

			logger := zerolog.New(w)

			ctx, cancel := context.WithCancel(logger.WithContext(t.Context()))
			defer cancel()

			source := servicesmocks.NewMockRequestEmailUpdateSource(t)

			if testCase.createShortCodeData != nil {
				source.EXPECT().
					CreateShortCode(ctx, services.CreateShortCodeRequest{
						Usage:    models.ShortCodeUsageValidateMail,
						Target:   testCase.request.ID.String(),
						TTL:      config.ShortCodes.Usages[models.ShortCodeUsageValidateMail].TTL,
						Data:     testCase.request.Email,
						Override: true,
					}).
					Return(testCase.createShortCodeData.resp, testCase.createShortCodeData.err)
			}

			service := services.NewRequestEmailUpdateService(source)

			resp, err := service.RequestEmailUpdate(ctx, testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)

			cancel()

			if err == nil {
				require.Equal(t, testCase.createShortCodeData.resp, resp)

				outC := make(chan []byte)
				// copy the output in a separate goroutine so printing can't block indefinitely
				go func() {
					var buf bytes.Buffer
					_, _ = io.Copy(&buf, r)
					outC <- buf.Bytes()
				}()

				service.Wait()
				require.NoError(t, w.Close())

				var jsonLog struct {
					Level               string         `json:"level"`
					DynamicTemplateData map[string]any `json:"dynamicTemplateData"`
				}

				out := <-outC
				require.NoError(t, json.Unmarshal(out, &jsonLog))

				require.Equal(t, "info", jsonLog.Level, string(out))
				require.Equal(t, map[string]any{
					"ShortCode": testCase.createShortCodeData.resp.PlainCode,
					"Target":    testCase.request.ID.String(),
					"URL":       config.SMTP.URLs.UpdateEmail,
					"Duration":  config.ShortCodes.Usages[models.ShortCodeUsageValidateMail].TTL.String(),
				}, jsonLog.DynamicTemplateData)
			}

			source.AssertExpectations(t)
		})
	}
}
