package services_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/context"

	"github.com/a-novel/authentication/config"
	"github.com/a-novel/authentication/internal/dao"
	"github.com/a-novel/authentication/internal/services"
	servicesmocks "github.com/a-novel/authentication/internal/services/mocks"
	"github.com/a-novel/authentication/models"
)

func TestRequestPasswordReset(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type selectCredentialsData struct {
		resp *dao.CredentialsEntity
		err  error
	}

	type createShortCodeData struct {
		resp *models.ShortCode
		err  error
	}

	testCases := []struct {
		name string

		request services.RequestPasswordResetRequest

		selectCredentialsData *selectCredentialsData
		createShortCodeData   *createShortCodeData

		expectErr error
	}{
		{
			name: "Success",

			request: services.RequestPasswordResetRequest{
				Email: "user@provider.com",
			},

			selectCredentialsData: &selectCredentialsData{
				resp: &dao.CredentialsEntity{
					ID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
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

			request: services.RequestPasswordResetRequest{
				Email: "user@provider.com",
			},

			selectCredentialsData: &selectCredentialsData{
				resp: &dao.CredentialsEntity{
					ID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
			},

			createShortCodeData: &createShortCodeData{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "SelectEmailError",

			request: services.RequestPasswordResetRequest{
				Email: "user@provider.com",
			},

			selectCredentialsData: &selectCredentialsData{
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

			source := servicesmocks.NewMockRequestPasswordResetSource(t)

			if testCase.createShortCodeData != nil {
				source.EXPECT().
					CreateShortCode(ctx, services.CreateShortCodeRequest{
						Usage:    models.ShortCodeUsageResetPassword,
						Target:   testCase.selectCredentialsData.resp.ID.String(),
						TTL:      config.ShortCodes.Usages[models.ShortCodeUsageResetPassword].TTL,
						Override: true,
					}).
					Return(testCase.createShortCodeData.resp, testCase.createShortCodeData.err)
			}

			if testCase.selectCredentialsData != nil {
				source.EXPECT().
					SelectCredentialsByEmail(ctx, testCase.request.Email).
					Return(testCase.selectCredentialsData.resp, testCase.selectCredentialsData.err)
			}

			service := services.NewRequestPasswordResetService(source)

			resp, err := service.RequestPasswordReset(ctx, testCase.request)
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
					"Target":    testCase.selectCredentialsData.resp.ID.String(),
					"URL":       config.SMTP.URLs.UpdatePassword,
					"Duration":  config.ShortCodes.Usages[models.ShortCodeUsageResetPassword].TTL.String(),
				}, jsonLog.DynamicTemplateData)
			}

			source.AssertExpectations(t)
		})
	}
}
